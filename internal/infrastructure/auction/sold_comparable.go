package auction

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1/yahoo_auctionv1connect"

	appmarket "jo3qma.com/yahoo_auctions_bot/internal/application/market"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

const (
	soldSearchMaxPages     = 3
	soldSearchMaxInspect   = 30
	soldSearchItemsPerPage = 20
)

// SoldComparableSearcher は落札済み Comparable の価格検索を行う。
type SoldComparableSearcher interface {
	SearchSoldPrices(ctx context.Context, category product.Category, identityKey, identityValue string, lookbackDays int) ([]int64, error)
}

// soldComparableClient は SearchAuctions + GetAuction による落札価格収集の実装。
// 専用 RPC 追加前の暫定実装。identityValue をキーワードに検索し FINISHED の落札価格を集める。
type soldComparableClient struct {
	base *client
}

// NewSoldComparableSearcher は SoldComparableSearcher を生成する。
func NewSoldComparableSearcher(baseURL string, httpClient *http.Client) SoldComparableSearcher {
	return &soldComparableClient{base: &client{httpClient: pickHTTPClient(httpClient), baseURL: baseURL}}
}

// WrapSoldComparableSearcher は既存 Client と同じ接続設定で SoldComparableSearcher を返す。
// Client が SoldComparableSearcher でも *client でもない場合は nil を返す。
func WrapSoldComparableSearcher(c Client) SoldComparableSearcher {
	if sc, ok := c.(SoldComparableSearcher); ok {
		return sc
	}
	if bc, ok := c.(*client); ok {
		return &soldComparableClient{base: bc}
	}
	return nil
}

func (c *soldComparableClient) SearchSoldPrices(ctx context.Context, category product.Category, identityKey, identityValue string, lookbackDays int) ([]int64, error) {
	_ = category
	_ = identityKey
	if strings.TrimSpace(identityValue) == "" {
		return nil, nil
	}

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	svc := yahoo_auctionv1connect.NewYahooAuctionServiceClient(&c.base.httpClient, c.base.baseURL)

	var prices []int64
	inspected := 0
	for page := int64(0); page < soldSearchMaxPages && inspected < soldSearchMaxInspect; page++ {
		resp, err := svc.SearchAuctions(ctx, connect.NewRequest(&yahoo_auctionv1.SearchAuctionsRequest{
			Query: identityValue,
			Page:  page,
		}))
		if err != nil {
			return nil, err
		}
		items := resp.Msg.GetItems()
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			if inspected >= soldSearchMaxInspect {
				break
			}
			inspected++
			data, err := c.base.GetAuction(ctx, item.AuctionId)
			if err != nil {
				log.Printf("[sold_comparable] GetAuction %s: %v", item.AuctionId, err)
				continue
			}
			if data.Status != yahoo_auctionv1.AuctionStatus_AUCTION_STATUS_FINISHED.String() {
				continue
			}
			if data.EndTime == nil || data.EndTime.Before(cutoff) {
				continue
			}
			if data.CurrentPrice > 0 {
				prices = append(prices, data.CurrentPrice)
			}
		}
		if int64(len(items)) < soldSearchItemsPerPage {
			break
		}
	}
	return prices, nil
}

var _ appmarket.SoldComparableSearcher = (*soldComparableClient)(nil)

func pickHTTPClient(h *http.Client) http.Client {
	if h != nil {
		return *h
	}
	return *http.DefaultClient
}
