output "route53_private_hosted_zone_ns" {
  value = aws_route53_zone.private_zone.name_servers
  description = "Values to add to the public hosted zone in dustin's account"
}