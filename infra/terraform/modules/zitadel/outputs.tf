output "url" {
  description = "Zitadel Cloud Run service URL"
  value       = google_cloud_run_v2_service.zitadel.uri
}

output "service_name" {
  description = "Zitadel Cloud Run service name"
  value       = google_cloud_run_v2_service.zitadel.name
}

output "service_account_email" {
  description = "Zitadel service account email"
  value       = google_service_account.zitadel.email
}
