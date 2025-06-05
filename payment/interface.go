package payment

import "paymentbot/models"

type PaymentLinkGenerator interface {
	GenerateLink(data *models.PaymentLinkData) (link string, paymentID string, err error)
}
