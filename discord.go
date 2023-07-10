package main

import (
	"context"
	"fmt"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/webhook"
	"github.com/disgoorg/snowflake/v2"
	"strconv"
	"strings"
	"time"
)

var WebhookClient webhook.Client

func CreateWebhookClient(webhookUrl string) {
	webhookArray := strings.Split(webhookUrl, "/")
	id, _ := strconv.ParseUint(webhookArray[len(webhookArray)-2], 10, 64)
	token := webhookArray[len(webhookArray)-1]

	WebhookClient = webhook.New(snowflake.ID(id), token)
}

func SendDmarketPurchase(product *DmarketProduct, marketType string, orderId string) {
	var embed = discord.NewEmbedBuilder()
	embed.SetTitle("Successful Purchase: " + orderId).SetURL("https://dmarket.com/ingame-items/item-list/csgo-skins")
	embed.SetTimestamp(time.Now()).SetThumbnail(product.Image)

	var description string
	if marketType == "dmarket" {
		description = "Purchased from the DMarket bot."
	} else {
		description = "Purchased P2P. SEND TRADE NOW."
	}
	embed.SetDescription(description)

	buffPrice := GetBuffPrice(product.Title, product.Extra.PhaseTitle)
	numPrice, _ := strconv.ParseFloat(product.Price.USD, 32)
	numPrice = numPrice / 100

	inline := true
	embed.SetColor(5763719)
	embed.SetFields(discord.EmbedField{
		Name:   "Price",
		Value:  fmt.Sprintf("$%.2f (%.2f%%)", numPrice, PercentageDifference(numPrice, buffPrice)),
		Inline: &inline,
	}, discord.EmbedField{
		Name:   "Buff Price",
		Value:  fmt.Sprintf("[$%.2f](%s)", buffPrice, GetBuffUrl(product.Title)),
		Inline: &inline,
	})

	WebhookClient.CreateEmbeds([]discord.Embed{embed.Build()})
}

func SendSkinportProduct(name string, image string, inspectLink string, price float64, purchaseUrl string, buffPrice float64, marketName string) {
	var embed = discord.NewEmbedBuilder()
	embed.SetTitle(marketName + ": " + name).SetURL(purchaseUrl)
	embed.SetTimestamp(time.Now()).SetThumbnail(image)

	inline := true
	embed.SetFields(discord.EmbedField{
		Name:   "Price",
		Value:  fmt.Sprintf("$%.2f (%.2f%%)", price, PercentageDifference(price, buffPrice)),
		Inline: &inline,
	}, discord.EmbedField{
		Name:   "Buff Price",
		Value:  fmt.Sprintf("[$%.2f](%s)", buffPrice, GetBuffUrl(name)),
		Inline: &inline,
	})

	WebhookClient.CreateEmbeds([]discord.Embed{embed.Build()})
}

func ReportATC() {
	webhookArray := strings.Split(config.Webhook, "/")
	id, _ := strconv.ParseUint(webhookArray[len(webhookArray)-2], 10, 64)
	token := webhookArray[len(webhookArray)-1]

	atcClient := webhook.New(snowflake.ID(id), token)
	atcClient.CreateMessage(discord.WebhookMessageCreate{Content: "Item ATCd: https://skinport.com/cart"})
}

func ReportError(err error) {
	WebhookClient.CreateMessage(discord.WebhookMessageCreate{Content: "Error encountered: " + err.Error()})
}

func GetUserInput(message string) string {
	input := ""
	client, err := disgo.New(config.BotToken,
		// set gateway options
		bot.WithGatewayConfigOpts(
			// set enabled intents
			gateway.WithIntents(
				gateway.IntentGuildMessages,
				gateway.IntentMessageContent,
			),
		),
		bot.WithEventListenerFunc(func(event *events.MessageCreate) {
			if event.Message.Author.Bot || event.Message.ChannelID.String() != config.InputChannel {
				return
			}
			input = event.Message.Content
		}),
	)
	if err != nil {
		panic(err)
	}

	// connect to the gateway
	if err = client.OpenGateway(context.TODO()); err != nil {
		panic(err)
	}

	channelId, _ := snowflake.Parse(config.InputChannel)
	client.Rest().CreateMessage(channelId, discord.NewMessageCreateBuilder().SetContent(message).Build())

	for true {
		if input != "" {
			break
		}
		time.Sleep(time.Second)
	}

	return input
}
