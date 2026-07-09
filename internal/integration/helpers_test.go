package integration

import (
	"io"
	"log/slog"

	invdomain "github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

func invDomainProductID(id string) invdomain.ProductID { return invdomain.ProductID(id) }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
