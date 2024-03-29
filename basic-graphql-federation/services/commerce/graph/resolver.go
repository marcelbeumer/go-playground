package graph

import (
	"fmt"
	"strconv"

	"github.com/marcelbeumer/go-playground/basic-graphql-federation/services/commerce/graph/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	TopProducts []*model.Product
}

func NewResolver() *Resolver {
	products := make([]*model.Product, 10)
	for x := 0; x < len(products); x++ {
		descr := fmt.Sprintf("12345_%d", x)
		products[x] = &model.Product{
			Sku:         fmt.Sprintf("12345_%d", x),
			Name:        strconv.Itoa(x),
			Price:       100,
			Description: &descr,
		}
	}
	return &Resolver{TopProducts: products}
}
