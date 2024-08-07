package githubutils

import (
	"context"
	"fmt"

	"github.com/google/go-github/v63/github"
)

// ErrBreak is returned by the fn callback to stop the [Range] operation.
var ErrBreak = fmt.Errorf("range operation stopped explicitly")

// Range calls fn for every item returned by req for all available pages.
func Range[T any](
	ctx context.Context,
	req func(context.Context, *github.ListOptions) ([]T, *github.Response, error),
	fn func(context.Context, T) error,
) error {
	return RangePages(
		ctx,
		req,
		func(ctx context.Context, items []T) error {
			for _, i := range items {
				if err := fn(ctx, i); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// RangePages calls fn for every page returned by req.
func RangePages[T any](
	ctx context.Context,
	req func(context.Context, *github.ListOptions) ([]T, *github.Response, error),
	fn func(context.Context, []T) error,
) error {
	opts := &github.ListOptions{
		Page: 1,
	}

	for opts.Page > 0 {
		items, res, err := req(ctx, opts)
		if err != nil {
			return err
		}

		switch err := fn(ctx, items); err {
		case nil:
			// continue
		case ErrBreak:
			return nil
		default:
			return err
		}

		opts.Page = res.NextPage
	}

	return nil
}
