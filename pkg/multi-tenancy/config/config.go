package config

import (
	"flag"
	"fmt"
	"strconv"
	"time"
)

// Config denotes the configuration required for multi-tenancy.
type Config struct {
	AuthKey string
	AuthValidDuration time.Duration
	ValidTenantsStr string
	ValidTenantsList []string // This gets filled in runtime.
}

func multiTenantFlags(fs *flag.FlagSet, cfg *Config) {
	fs.StringVar(&cfg.AuthKey, "multi-tenancy-auth-key", "", "Set authorization key for multi-tenancy. " +
		"Authorization in multi-tenancy is implemented by JWT. The authorization key set by 'multi-tenant-auth-key' will " +
		"be used as a signature during the formation of JWT.")
	fs.DurationVar(&cfg.AuthValidDuration, "multi-tenancy-auth-duration", 0, "Set duration after which " +
		"the JWT would be considered as invalid. Setting this to 0 (default) would mean that the token will never expire.")
	fs.StringVar(&cfg.ValidTenantsStr, "multi-tenancy-valid-tenants", "[]", "List of tenant names separated " +
		"by commas. Tenants mentioned in this list will be respect as actual tenants, while reading or writing data. Writing/Reading " +
		"operations will be invalid on those tenants that are not mentioned. Setting this to '[]' will allow write-request " +
		"from all tenants.")
}

// ContainsAuthKey returns true if auth key is found in multi-tenant config.
func (mtc *Config) ContainsAuthKey() bool {
	return mtc.AuthKey != ""
}

// FillTenantList fills the ValidTenantsList from ValidTenantStr.
func (mtc *Config) FillTenantList() error {
	if strconv.Itoa(int(mtc.ValidTenantsStr[0])) != "[" || strconv.Itoa(int(mtc.ValidTenantsStr[len(mtc.ValidTenantsStr)-1])) != "]" {
		return fmt.Errorf("invalid tenants list. ")
	}
}
