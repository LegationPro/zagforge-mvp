environment          = "staging"
github_repo          = "LegationPro/zagforge-cloud"
project_id           = "zagforge"
region               = "europe-west1"
database_provider    = "cloudsql"
cloud_sql_tier       = "db-custom-1-3840"
redis_provider       = "upstash"
api_min_instances    = 0
api_max_instances    = 4
cloud_armor_enabled  = false
github_app_id        = "3122231"
github_app_slug      = "zagforge-zigzag-dev"
cors_allowed_origins = ""

# Zitadel
zitadel_domain        = "auth-staging.zagforge.com"
zitadel_min_instances = 0
zitadel_max_instances = 2
