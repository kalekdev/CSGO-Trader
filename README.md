# CSGO Trader
A Counter Strike skin arbitrage trading application written in Golang - supports buying on [Dmarket](https://dmarket.com/) and [Skinport](https://skinport.com/) and selling on [Buff 163](https://buff.163.com/).

## Setup:
1. [Install Golang](https://go.dev/dl)
2. Complete `config.json` with the desired values
3. Navigate to the source directory
4. Execute `go mod download` to install the required packages
5. Execute `go run main.go` to start the program

## `config.json` values
* `monitorDelay` - Delay in ms between checking for new Dmarket products (5000-10000 recommended to avoid rate limits)
* `webhook` - URL of the Discord webhook where you'd like to receive add to cart and purchase notifications
* `skinportUsername` & `skinportPassword` - Your Skinport account details
* `twoCaptchaKey` - Your 2captcha key used for solving Skinport captchas (not needed if logging in manually)
* `botToken` - Discord bot token for collecting Skinport email confirmation links / auth token through Discord
* `inputChannel` - Discord channel ID of the channel in which you want to send Skinport details
* `dmarketPublicKey` & `dmarketPrivateKey` - [Dmarket API](https://dmarket.com/blog/dmarket-api-for-automated-trading/#API-section) details

## Issues
Skinport automated login and order submission very rarely works due to the v3 captcha they have introduced, which is why I decided to open source the project. Instead, it will now send notifications after adding a desired item to cart, from which the user can complete checkout manually.

For the manual Skinport login, the user must first login in the browser and copy their `connect.sid` cookie, which the Discord bot will ask for after starting the program. This cookie usually lasts a week but will need to be entered whenever the program restarts. 

## Contributing
I was able to make a good amount of money using this program in the run up to CS2 (mainly from buying on Dmarket). However, there are several features I have in mind that would improve the project. Please feel free to contribute or suggest any improvements:
* Fixing the Skinport captcha issue, perhaps by creating a local captcha harvester.
* Persisting purchased items to a database, making selling and profit calculations easier.
* Automate selling on Buff, this would allow the program to generate profit without any manual work.