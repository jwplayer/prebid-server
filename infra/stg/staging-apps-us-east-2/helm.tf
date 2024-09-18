module "helm-release-cluster" {
  source        = "gitlab.ops.connatix.com/connatix/helm-release/deploy"
  chart_repo    = "http://chartmuseum.ops.connatix.com/"
  chart_version = "^11.0.0"
  chart_name    = "cnx-universal-chart"
  release_name  = "prebid-server"
  values        = ["${file("helm/values.yaml")}"]

  overrides = {
    "containerSettings.image.tag" : var.cnx_version
    "project.url":      var.cnx_project_url
    "containerSettings.image.repository" : var.cnx_image_repo
  }
}
