package options

import "errors"

// Validate validates the options
// 验证选项
func (o *Options) Validate() error {
	var errs []error

	errs = append(errs, o.ServerOptions.Validate())
	errs = append(errs, o.DatabaseOptions.Validate())
	errs = append(errs, o.RedisOptions.Validate())

	return errors.Join(errs...)
}
