resource "aws_dynamodb_table" "orders" {
  name         = local.table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  global_secondary_index {
    name            = local.status_index
    hash_key        = "status"
    projection_type = "ALL"
  }

  tags = local.common_tags
}
