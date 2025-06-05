package payment

import "paymentbot/models"

type PaymentLinkGenerator interface {
	GenerateLink(data *models.PaymentLinkData) (string, error)
}
