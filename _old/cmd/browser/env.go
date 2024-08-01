package main

import "github.com/dogmatiq/ferrite"

var githubAppID = ferrite.
	Unsigned[uint]("GITHUB_APP_ID", "the ID of the GitHub application used to read repository content").
	WithMinimum(1).
	Required()

var githubAppClientID = ferrite.
	String("GITHUB_CLIENT_ID", "the client ID of the GitHub application used to read repository content").
	Required()

var githubAppClientSecret = ferrite.
	String("GITHUB_CLIENT_SECRET", "the client secret for the GitHub application used to read repository content").
	WithSensitiveContent().
	Required()

var githubAppPrivateKey = ferrite.
	String("GITHUB_APP_PRIVATEKEY", "the private key for the GitHub application used to read repository content").
	WithSensitiveContent().
	Required()

var githubAppHookSecret = ferrite.
	String("GITHUB_HOOK_SECRET", "the secret used to verify GitHub web-hook requests are genuine").
	WithSensitiveContent().
	Required()

var githubURL = ferrite.
	URL("GITHUB_URL", "the base URL of the GitHub API").
	Optional()

var postgresDSN = ferrite.
	String("DSN", "the PostgreSQL connection string").
	Required()
