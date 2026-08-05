package persist

import "errors"

var ErrInsufficientQty = errors.New("insufficient inventory qty")

// SpendInventory decrements qty; returns ErrInsufficientQty if stock is too low.
func (s *Store) SpendInventory(playerID, sku string, n int) error {
	if n < 1 {
		return nil
	}
	return s.mutate(func(d *db) error {
		k := invKey(playerID, sku)
		it := d.Inventory[k]
		if it.Qty < n {
			return ErrInsufficientQty
		}
		it.PlayerID = playerID
		it.SKU = sku
		it.Qty -= n
		d.Inventory[k] = it
		return nil
	})
}
