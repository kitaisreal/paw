#!/bin/bash

command -v go >/dev/null 2>&1 || { echo "go is not installed"; exit 1; }
# command -v gosec >/dev/null 2>&1 || { echo "gosec is not installed"; exit 1; }
command -v staticcheck >/dev/null 2>&1 || { echo "staticcheck is not installed"; exit 1; }
command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is not installed"; exit 1; }

echo "Running go vet..."
go_vet_output=$(go vet ./... 2>&1)
go_vet_status=$?

# echo "Running gosec..."
# gosec_output=$(gosec ./... 2>&1)
# gosec_status=$?

echo "Running staticcheck..."
staticcheck_output=$(staticcheck ./... 2>&1)
staticcheck_status=$?

echo "Running golangci-lint..."
golangci_output=$(golangci-lint run ./... 2>&1)
golangci_status=$?

echo "=============== go vet results ==============="
echo "$go_vet_output"
echo "Exit code: $go_vet_status"
echo

# echo "=============== gosec results ==============="
# echo "$gosec_output"
# echo "Exit code: $gosec_status"
# echo

echo "=============== staticcheck results ==============="
echo "$staticcheck_output"
echo "Exit code: $staticcheck_status"
echo

echo "=============== golangci-lint results ==============="
echo "$golangci_output"
echo "Exit code: $golangci_status"
echo

failed_linters=""
if [ $go_vet_status -ne 0 ]; then
    failed_linters="go vet"
fi
# if [ $gosec_status -ne 0 ]; then
#     if [ -n "$failed_linters" ]; then
#         failed_linters="$failed_linters, "
#     fi
#     failed_linters="${failed_linters}gosec"
# fi
if [ $staticcheck_status -ne 0 ]; then
    if [ -n "$failed_linters" ]; then
        failed_linters="$failed_linters, "
    fi
    failed_linters="${failed_linters}staticcheck"
fi
if [ $golangci_status -ne 0 ]; then
    if [ -n "$failed_linters" ]; then
        failed_linters="$failed_linters, "
    fi
    failed_linters="${failed_linters}golangci-lint"
fi

if [ -n "$failed_linters" ]; then
    echo "The following linters failed: $failed_linters"
    exit 1
fi

echo "All linters passed"
