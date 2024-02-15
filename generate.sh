#!/bin/bash

cert_C="SI"
cert_ST="Slovenia"
cert_L="Ljubljana"
cert_O="My Test Org"
cert_OU="Dev"
cert_CN="Test Certificates"

folder_name="./certs"
cert_name="cert.pem"
key_name="key.pem"
common_name="127.0.0.1"
openssl req -x509 -newkey rsa:4096 -keyout ./certs/key.pem -out ./certs/cert.pem \
    -sha256 -days 3650 -nodes -subj \
    "/C=${cert_C}/ST=${cert_ST}/L=${cert_L}/O=${cert_O}/OU=${cert_OU}/CN=${common_name}"
chmod 666 ./certs/*
