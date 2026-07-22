#!/bin/bash

# Test script to verify range extraction implementation

echo "=== Range Extraction Verification Test ==="

# Test 1: DOCX range extraction
echo "Test 1: DOCX range extraction (2-4)"
./clip gpt.docx 2-4
if [ $? -eq 0 ]; then
    echo "✅ PASS: DOCX range extraction works"
else
    echo "❌ FAIL: DOCX range extraction failed"
fi
echo

# Test 2: TXT range extraction
echo "Test 2: TXT range extraction (1-3)"
./clip test.txt 1-3
if [ $? -eq 0 ]; then
    echo "✅ PASS: TXT range extraction works"
else
    echo "❌ FAIL: TXT range extraction failed"
fi
echo

# Test 3: CSV range extraction
echo "Test 3: CSV range extraction (1-2)"
./clip test.csv 1-2
if [ $? -eq 0 ]; then
    echo "✅ PASS: CSV range extraction works"
else
    echo "❌ FAIL: CSV range extraction failed"
fi
echo

# Test 4: Full document extraction (no range)
echo "Test 4: Full DOCX extraction (no range)"
./clip gpt.docx
if [ $? -eq 0 ]; then
    echo "✅ PASS: Full document extraction works"
else
    echo "❌ FAIL: Full document extraction failed"
fi
echo

# Test 5: Verify no PDF-only restriction error
echo "Test 5: Verify no 'page ranges are only supported for PDF files' error"
output=$(./clip gpt.docx 1-2 2>&1)
if echo "$output" | grep -q "page ranges are only supported for PDF files"; then
    echo "❌ FAIL: PDF-only restriction still exists"
    echo "Error output: $output"
else
    echo "✅ PASS: No PDF-only restriction found"
fi
echo

# Test 6: Range validation (out of bounds)
echo "Test 6: Range validation (out of bounds)"
output=$(./clip gpt.docx 5-10 2>&1)
if echo "$output" | grep -q "exceeds document paragraph count"; then
    echo "✅ PASS: Range validation works correctly"
else
    echo "❌ FAIL: Range validation not working"
    echo "Output: $output"
fi
echo

echo "=== All tests completed ==="