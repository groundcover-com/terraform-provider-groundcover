# Policies are imported by their UUID, not their name.
# Find the UUID under Settings → Policies (visible in the network tab of the policy list request),
# or copy it from `groundcover_policy.<name>.uuid` after creating one with Terraform.
terraform import groundcover_policy.example "00000000-0000-0000-0000-000000000000"
