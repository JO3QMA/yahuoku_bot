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

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

type soldStubHandler struct {
	searchItems []*yahoo_auctionv1.SearchAuctionsResponse_Item
	getAuction  func(auctionID string) (*yahoo_auctionv1.GetAuctionResponse, error)
}

func (s *soldStubHandler) GetAuction(_ context.Context, req *connect.Request[yahoo_auctionv1.GetAuctionRequest]) (*connect.Response[yahoo_auctionv1.GetAuctionResponse], error) {
	if s.getAuction != nil {
		resp, err := s.getAuction(req.Msg.AuctionId)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(resp), nil
	}
	return nil, connect.NewError(connect.CodeInternal, nil)
}

func (s *soldStubHandler) GetCategoryItems(context.Context, *connect.Request[yahoo_auctionv1.GetCategoryItemsRequest]) (*connect.Response[yahoo_auctionv1.GetCategoryItemsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *soldStubHandler) SearchAuctions(context.Context, *connect.Request[yahoo_auctionv1.SearchAuctionsRequest]) (*connect.Response[yahoo_auctionv1.SearchAuctionsResponse], error) {
	return connect.NewResponse(&yahoo_auctionv1.SearchAuctionsResponse{Items: s.searchItems}), nil
}

func newSoldSearcher(t *testing.T, h *soldStubHandler) SoldComparableSearcher {
	t.Helper()
	mux := http.NewServeMux()
	path, hh := yahoo_auctionv1connect.NewYahooAuctionServiceHandler(h)
	mux.Handle(path, hh)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewSoldComparableSearcher(srv.URL, srv.Client())
}

func TestSearchSoldPrices_emptyIdentity(t *testing.T) {
	sc := newSoldSearcher(t, &soldStubHandler{})
	prices, err := sc.SearchSoldPrices(context.Background(), product.CategoryGPU, "model", "  ", 90)
	if err != nil || prices != nil {
		t.Fatalf("got prices=%v err=%v", prices, err)
	}
}

func TestSearchSoldPrices_noSearchResults(t *testing.T) {
	sc := newSoldSearcher(t, &soldStubHandler{searchItems: nil})
	prices, err := sc.SearchSoldPrices(context.Background(), product.CategoryGPU, "model", "GTX1080", 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 0 {
		t.Fatalf("got %v", prices)
	}
}

func TestSearchSoldPrices_skipsNonFinished(t *testing.T) {
	sc := newSoldSearcher(t, &soldStubHandler{
		searchItems: []*yahoo_auctionv1.SearchAuctionsResponse_Item{{AuctionId: "a1"}},
		getAuction: func(string) (*yahoo_auctionv1.GetAuctionResponse, error) {
			return &yahoo_auctionv1.GetAuctionResponse{
				Status:       yahoo_auctionv1.AuctionStatus_AUCTION_STATUS_ACTIVE,
				CurrentPrice: 5000,
				AuctionInformation: &yahoo_auctionv1.AuctionInformation{
					EndTime: timestamppb.New(time.Now()),
				},
			}, nil
		},
	})
	prices, err := sc.SearchSoldPrices(context.Background(), product.CategoryGPU, "model", "GTX1080", 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 0 {
		t.Fatalf("got %v", prices)
	}
}

func TestSearchSoldPrices_skipsBeforeCutoff(t *testing.T) {
	sc := newSoldSearcher(t, &soldStubHandler{
		searchItems: []*yahoo_auctionv1.SearchAuctionsResponse_Item{{AuctionId: "a1"}},
		getAuction: func(string) (*yahoo_auctionv1.GetAuctionResponse, error) {
			return &yahoo_auctionv1.GetAuctionResponse{
				Status:       yahoo_auctionv1.AuctionStatus_AUCTION_STATUS_FINISHED,
				CurrentPrice: 7000,
				AuctionInformation: &yahoo_auctionv1.AuctionInformation{
					EndTime: timestamppb.New(time.Now().AddDate(0, 0, -200)),
				},
			}, nil
		},
	})
	prices, err := sc.SearchSoldPrices(context.Background(), product.CategoryGPU, "model", "GTX1080", 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 0 {
		t.Fatalf("got %v", prices)
	}
}

func TestSearchSoldPrices_collectsFinished(t *testing.T) {
	sc := newSoldSearcher(t, &soldStubHandler{
		searchItems: []*yahoo_auctionv1.SearchAuctionsResponse_Item{{AuctionId: "a1"}},
		getAuction: func(string) (*yahoo_auctionv1.GetAuctionResponse, error) {
			return &yahoo_auctionv1.GetAuctionResponse{
				Status:       yahoo_auctionv1.AuctionStatus_AUCTION_STATUS_FINISHED,
				CurrentPrice: 9000,
				AuctionInformation: &yahoo_auctionv1.AuctionInformation{
					EndTime: timestamppb.New(time.Now().AddDate(0, 0, -1)),
				},
			}, nil
		},
	})
	prices, err := sc.SearchSoldPrices(context.Background(), product.CategoryGPU, "model", "GTX1080", 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 1 || prices[0] != 9000 {
		t.Fatalf("got %v", prices)
	}
}

func TestSearchSoldPrices_allGetAuctionFail(t *testing.T) {
	sc := newSoldSearcher(t, &soldStubHandler{
		searchItems: []*yahoo_auctionv1.SearchAuctionsResponse_Item{{AuctionId: "a1"}},
		getAuction: func(string) (*yahoo_auctionv1.GetAuctionResponse, error) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		},
	})
	_, err := sc.SearchSoldPrices(context.Background(), product.CategoryGPU, "model", "GTX1080", 90)
	if err == nil {
		t.Fatal("expected error")
	}
}
