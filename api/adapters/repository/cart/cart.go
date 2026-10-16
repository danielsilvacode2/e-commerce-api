package cart

import (
	"database/sql"
	entities "e-commerce/core/entities/cart"
)

type cartRepository struct {
	db *sql.DB
}

func NewCartRepository(db *sql.DB) entities.CartRepository{
	return &cartRepository{
		db: db,
	}
}

func (r *cartRepository )CreateCart(entities.Cart) error {
	
	return nil
}


func (r *cartRepository )	ReadCartByUserId(userId string) (*entities.Cart, error){
	return nil, nil
}