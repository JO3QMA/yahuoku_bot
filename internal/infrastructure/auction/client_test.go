package auction

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1/yahoo_auctionv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stubHandler struct {
	err   error
	noEnd bool
}

func (s *stubHandler) GetAuction(ctx context.Context, req *connect.Request[yahoo_auctionv1.GetAuctionRequest]) (*connect.Response[yahoo_auctionv1.GetAuctionResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	end := timestamppb.New(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	resp := &yahoo_auctionv1.GetAuctionResponse{
		AuctionId:    req.Msg.AuctionId,
		Title:        "t",
		CurrentPrice: 100,
		Status:       yahoo_auctionv1.AuctionStatus_AUCTION_STATUS_ACTIVE,
		Images:       []string{"https://img.example/x.png"},
		Description:  "d",
	}
	if !s.noEnd {
		resp.AuctionInformation = &yahoo_auctionv1.AuctionInformation{
			AuctionId: req.Msg.AuctionId,
			EndTime:   end,
		}
	}
	return connect.NewResponse(resp), nil
}

func (s *stubHandler) GetCategoryItems(context.Context, *connect.Request[yahoo_auctionv1.GetCategoryItemsRequest]) (*connect.Response[yahoo_auctionv1.GetCategoryItemsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *stubHandler) SearchAuctions(context.Context, *connect.Request[yahoo_auctionv1.SearchAuctionsRequest]) (*connect.Response[yahoo_auctionv1.SearchAuctionsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestNewClient_customHTTPClient(t *testing.T) {
	h := &stubHandler{}
	mux := http.NewServeMux()
	path, hh := yahoo_auctionv1connect.NewYahooAuctionServiceHandler(h)
	mux.Handle(path, hh)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	hc := srv.Client()
	c := NewClient(srv.URL, hc)
	data, err := c.GetAuction(context.Background(), "abc12345678")
	if err != nil {
		t.Fatal(err)
	}
	if data.AuctionID != "abc12345678" || data.Title != "t" {
		t.Fatalf("%+v", data)
	}
	if data.EndTime == nil {
		t.Fatal("expected EndTime")
	}
}

func TestNewClient_defaultHTTPClient(t *testing.T) {
	h := &stubHandler{noEnd: true}
	mux := http.NewServeMux()
	path, hh := yahoo_auctionv1connect.NewYahooAuctionServiceHandler(h)
	mux.Handle(path, hh)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL, nil)
	data, err := c.GetAuction(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if data.EndTime != nil {
		t.Fatal("expected nil EndTime")
	}
}

func TestClient_GetAuction_error(t *testing.T) {
	h := &stubHandler{err: connect.NewError(connect.CodeNotFound, nil)}
	mux := http.NewServeMux()
	path, hh := yahoo_auctionv1connect.NewYahooAuctionServiceHandler(h)
	mux.Handle(path, hh)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL, srv.Client())
	_, err := c.GetAuction(context.Background(), "id")
	if err == nil {
		t.Fatal("expected error")
	}
}
