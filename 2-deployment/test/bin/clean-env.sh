#!/bin/bash
kind delete cluster --name e2e
kubectl config delete-user e2e-context
kubectl config delete-cluster e2e-context
kubectl config delete-context e2e-context