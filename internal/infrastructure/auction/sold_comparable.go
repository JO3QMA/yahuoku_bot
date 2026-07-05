package auction

import (
	"context"
	"fmt"
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
func WrapSoldComparableSearcher(c Client) (SoldComparableSearcher, error) {
	if sc, ok := c.(SoldComparableSearcher); ok {
		return sc, nil
	}
	if bc, ok := c.(*client); ok {
		return &soldComparableClient{base: bc}, nil
	}
	return nil, fmt.Errorf("unsupported client type: %T", c)
}

func (c *soldComparableClient) SearchSoldPrices(ctx context.Context, category product.Category, identityKey, identityValue string, lookbackDays int) ([]int64, error) {
	// TODO: identityKey を SearchAuctions のフィルタに反映する（専用 RPC 追加時に実施）
	_ = identityKey
	if strings.TrimSpace(identityValue) == "" {
		return nil, nil
	}

	query := strings.TrimSpace(identityValue)
	if cat := category.DisplayName(); cat != "" {
		query = query + " " + cat
	}

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	svc := yahoo_auctionv1connect.NewYahooAuctionServiceClient(&c.base.httpClient, c.base.baseURL)

	var prices []int64
	inspected := 0
	getAuctionErrors := 0
	for page := int64(0); page < soldSearchMaxPages && inspected < soldSearchMaxInspect; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := svc.SearchAuctions(ctx, connect.NewRequest(&yahoo_auctionv1.SearchAuctionsRequest{
			Query: query,
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
				getAuctionErrors++
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
		// NOTE: API のページサイズが soldSearchItemsPerPage と一致する前提。
		// 将来的に SearchAuctionsResponse の total_pages / next_page_token 対応が望ましい。
		if int64(len(items)) < soldSearchItemsPerPage {
			break
		}
	}
	if len(prices) == 0 && getAuctionErrors > 0 && getAuctionErrors == inspected {
		return nil, fmt.Errorf("all GetAuction failed (%d items)", getAuctionErrors)
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
