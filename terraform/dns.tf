# --- Subdomain hosted zone ---

resource "aws_route53_zone" "demo" {
  name = local.domain

  tags = local.common_tags
}

# --- NS delegation from parent zone ---

resource "aws_route53_record" "delegation" {
  zone_id = data.aws_route53_zone.parent.zone_id
  name    = local.domain
  type    = "NS"
  ttl     = 300
  records = aws_route53_zone.demo.name_servers
}

# --- ACM wildcard certificate ---

resource "aws_acm_certificate" "wildcard" {
  domain_name       = "*.${local.domain}"
  validation_method = "DNS"

  tags = local.common_tags

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.wildcard.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  zone_id = aws_route53_zone.demo.zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 60
  records = [each.value.record]

  allow_overwrite = true
}

resource "aws_acm_certificate_validation" "wildcard" {
  certificate_arn         = aws_acm_certificate.wildcard.arn
  validation_record_fqdns = [for r in aws_route53_record.cert_validation : r.fqdn]
}

# --- App DNS record ---

resource "aws_route53_record" "app" {
  zone_id = aws_route53_zone.demo.zone_id
  name    = local.app_domain
  type    = "A"

  alias {
    name                   = aws_lb.main.dns_name
    zone_id                = aws_lb.main.zone_id
    evaluate_target_health = true
  }
}

# --- Observer DNS record ---

resource "aws_route53_record" "observer" {
  zone_id = aws_route53_zone.demo.zone_id
  name    = local.observer_domain
  type    = "A"

  alias {
    name                   = aws_lb.main.dns_name
    zone_id                = aws_lb.main.zone_id
    evaluate_target_health = true
  }
}
