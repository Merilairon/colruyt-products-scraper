package shared

type Price struct {
	BasicPrice           float32 `json:"basicPrice"`
	IsRedPrice           bool    `json:"isRedPrice"`
	MeasurementUnit      string  `json:"measurementUnit"`
	MeasurementUnitPrice float32 `json:"measurementUnitPrice"`
	RecommendedQuantity  string  `json:"recommendedQuantity"`
}

var PriceColumns = []string{
	"product_id",
	"basic_price",
	"is_red_price",
	"in_promo",
	"in_pre_condition_promo",
	"is_price_available",
	"measurement_unit",
	"measurement_unit_price",
	"recommended_quantity",
	"time",
	"promotion",
	"promo_codes",
}
