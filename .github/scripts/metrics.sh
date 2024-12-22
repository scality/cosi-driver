#!/bin/bash

# Declare expected values for each metric as environment variables
EXPECTED_CREATE_BUCKET=0
EXPECTED_DELETE_BUCKET=0
EXPECTED_GET_INFO=3
EXPECTED_GRANT_ACCESS=0
EXPECTED_REVOKE_ACCESS=0
GRPC_METHOD_TO_TEST="grpc_server_msg_sent_total"

# Fetch metrics and filter for grpc_server_msg_sent_total
METRICS=$(curl -s localhost:8080/metrics | grep $GRPC_METHOD_TO_TEST)
echo "Validating gRPC Server Metrics..."

# Loop through each line
echo "$METRICS" | while read -r line; do
  # Extract the grpc_method and value
  method=$(echo "$line" | sed -n 's/.*grpc_method="\([^"]*\)".*/\1/p') # Extract method name
  value=$(echo "$line" | awk '{print $NF}')                           # Extract value

  # Determine the expected value based on the grpc_method
  case "$method" in
    "DriverCreateBucket")
      expected_value=$EXPECTED_CREATE_BUCKET
      ;;
    "DriverDeleteBucket")
      expected_value=$EXPECTED_DELETE_BUCKET
      ;;
    "DriverGetInfo")
      expected_value=$EXPECTED_GET_INFO
      ;;
    "DriverGrantBucketAccess")
      expected_value=$EXPECTED_GRANT_ACCESS
      ;;
    "DriverRevokeBucketAccess")
      expected_value=$EXPECTED_REVOKE_ACCESS
      ;;
    *)
      echo "Unknown method: $method. Skipping validation."
      continue
      ;;
  esac

  # Display method, value, and expected value
  echo "Method: $method, Value: $value, Expected: $expected_value"

  # Perform validation
  if [[ "$value" -ne "$expected_value" ]]; then
    echo "Error: $method has an unexpected value ($value). Expected: $expected_value"
    exit 1
  fi
done

echo "All metrics validated successfully."
