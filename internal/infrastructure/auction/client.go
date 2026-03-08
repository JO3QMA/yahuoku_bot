package auction

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1/yahoo_auctionv1connect"
)

// AuctionData はGetAuction RPCのレスポンスをアプリケーション向けに変換したデータ。
type AuctionData struct {
	AuctionID    string
	Title        string
	CurrentPrice int64
	Status       string // ACTIVE / FINISHED / CANCELED
	Images       []string
	Description  string
	EndTime      *time.Time
}

// Client はyahoo_auctions APIへのConnect RPCクライアント。
type Client interface {
	GetAuction(ctx context.Context, auctionID string) (*AuctionData, error)
}

// client はClientの実装。
type client struct {
	httpClient http.Client
	baseURL    string
}

// NewClient はConnect RPCクライアントを生成する。
func NewClient(baseURL string, httpClient *http.Client) Client {
	hc := http.DefaultClient
	if httpClient != nil {
		hc = httpClient
	}
	return &client{httpClient: *hc, baseURL: baseURL}
}

// GetAuction はオークションIDから商品情報を取得する。
func (c *client) GetAuction(ctx context.Context, auctionID string) (*AuctionData, error) {
	svc := yahoo_auctionv1connect.NewYahooAuctionServiceClient(&c.httpClient, c.baseURL)
	resp, err := svc.GetAuction(ctx, connect.NewRequest(&yahoo_auctionv1.GetAuctionRequest{
		AuctionId: auctionID,
	}))
	if err != nil {
		return nil, err
	}

	r := resp.Msg
	data := &AuctionData{
		AuctionID:    r.AuctionId,
		Title:        r.Title,
		CurrentPrice: r.CurrentPrice,
		Status:       r.Status.String(),
		Images:       r.Images,
		Description:  r.Description,
	}

	if r.AuctionInformation != nil && r.AuctionInformation.EndTime != nil {
		t := r.AuctionInformation.EndTime.AsTime()
		data.EndTime = &t
	}

	return data, nil
}
