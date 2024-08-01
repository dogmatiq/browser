package web

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dogmatiq/browser/githubx"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func handleOAuthCallback(version string, c *oauth2.Config, key *rsa.PublicKey) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		oauthToken, err := c.Exchange(ctx, ctx.Query("code"))
		if err != nil {
			fmt.Println("unable to perform oauth exchange:", err) // TODO
			renderError(ctx, version, http.StatusUnauthorized)
			return
		}

		oauthTokenJSON, err := json.Marshal(oauthToken)
		if err != nil {
			fmt.Println("unable to marshal oauth token:", err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			ctx.Abort()
			return
		}

		opts := &jose.EncrypterOptions{}
		opts.WithType("JWT")
		opts.WithContentType("JWT")

		enc, err := jose.NewEncrypter(
			jose.A128CBC_HS256,
			jose.Recipient{
				Algorithm: jose.RSA_OAEP_256,
				Key:       key,
			},
			opts,
		)
		if err != nil {
			fmt.Println("unable to create token encrypter:", err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			ctx.Abort()
			return
		}

		claims := jwt.Claims{
			Issuer:  "dogmatiq/browser",
			Subject: string(oauthTokenJSON),
		}
		token, err := jwt.Encrypted(enc).Claims(claims).CompactSerialize()
		if err != nil {
			fmt.Println("unable to encrypt token:", err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			ctx.Abort()
			return
		}

		ctx.SetCookie(
			"token",
			token,
			int(time.Until(oauthToken.Expiry).Seconds()),
			"",
			"",
			true,
			true,
		)

		ctx.Redirect(http.StatusTemporaryRedirect, ctx.Query("state"))
	}
}

func requireOAuth(version string, c *githubx.Connector, key *rsa.PrivateKey) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := ctx.Cookie("token")
		if err != nil {
			redirectToLogin(ctx, c.OAuthConfig)
			return
		}

		parsedJWT, err := jwt.ParseEncrypted(token)
		if err != nil {
			fmt.Println("unable to parse encrypted token:", err) // TODO
			redirectToLogin(ctx, c.OAuthConfig)
			return
		}

		var claims jwt.Claims
		if err := parsedJWT.Claims(key, &claims); err != nil {
			fmt.Println("unable to decrypt claims:", err) // TODO
			redirectToLogin(ctx, c.OAuthConfig)
			return
		}

		var oauthToken oauth2.Token
		if err := json.Unmarshal([]byte(claims.Subject), &oauthToken); err != nil {
			fmt.Println("unable to unmarshal oauth token:", err) // TODO
			redirectToLogin(ctx, c.OAuthConfig)
			return
		}

		client, err := c.UserClient(ctx, &oauthToken)
		if err != nil {
			fmt.Println("unable to create user client:", err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			ctx.Abort()
		}

		user, res, err := client.Users.Get(ctx, "")
		if err != nil {
			if res != nil && res.StatusCode == http.StatusUnauthorized {
				redirectToLogin(ctx, c.OAuthConfig)
				return
			}

			fmt.Println("unable to query connected user:", err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			ctx.Abort()
			return
		}

		ctx.Set("github-user", user)
	}
}

func redirectToLogin(ctx *gin.Context, c *oauth2.Config) {
	authURL := c.AuthCodeURL(ctx.Request.URL.String())
	ctx.Redirect(http.StatusTemporaryRedirect, authURL)
	ctx.Abort()
}

func currentUser(ctx *gin.Context) (*github.User, bool) {
	u, ok := ctx.Value("github-user").(*github.User)
	return u, ok
}
