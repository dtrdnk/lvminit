#!/usr/bin/env bats

@test "Deploy lvminit via helm chart" {
  run kubectl create namespace lvminit || true
  run helm upgrade --install --namespace lvminit lvminit /helm --wait --values values.yaml
  run kubectl -n lvminit get pod
  echo "$output"
  [ "$status" -eq 0 ]
  sleep 5
}

@test "Get lvminit pods" {
  run kubectl -n lvminit get pod
  echo "$output"
  [ "$status" -eq 0 ]
  sleep 5
}

@test "lvminit DaemonSet should roll out" {
  run kubectl --namespace lvminit rollout status ds/lvminit --timeout=120s
  [ "$status" -eq 0 ]
  sleep 15
}

@test "LVM should see all loop PVs" {
  pod=$(kubectl --namespace lvminit get pod -l app.kubernetes.io/name=lvminit -o jsonpath='{.items[0].metadata.name}')
  run kubectl --namespace lvminit exec "$pod" -- lvm pvs --noheadings -o pv_name
  [ "$status" -eq 0 ]
  echo "LVM Physical Volumes detected:"
  echo "$output"
  # Check all disks are present as PVs
  for disk in $DISK_PATHS; do
    echo "$output" | grep "$disk"
  done
}

@test "LVM should have a volume group with all disks" {
  pod=$(kubectl --namespace lvminit get pod -l app.kubernetes.io/name=lvminit -o jsonpath='{.items[0].metadata.name}')
  run kubectl --namespace lvminit exec "$pod" -- lvm vgs --noheadings -o vg_name,pv_count
  [ "$status" -eq 0 ]
  # Limit check: vg present, pv_count >= DISK_COUNT
  echo "LVM Volume Groups found:"
  echo "$output"
  # TODO: fix this hardcoded vg name
  vg_count=$(echo "$output" | grep vgdata1 | awk '{print $2}' | head -1)
  # TODO: fix hardcoded value of disks
  [ "$vg_count" -ge 2 ]
}

@test "Destroy VG and PV via lvminit mode: destroy" {

  # Helm upgrade with destroy mode (make sure this cleans up)
  run helm upgrade --install --namespace lvminit lvminit /helm --wait --values values.yaml --values destroy-values.yaml
  [ "$status" -eq 0 ]

  # Wait for all VGs and PVs to disappear, give some time
  sleep 8

  pod=$(kubectl --namespace lvminit get pod -l app.kubernetes.io/name=lvminit -o jsonpath='{.items[0].metadata.name}')
  # VG should not exist
  run kubectl --namespace lvminit exec "$pod" -- lvm vgs --noheadings -o vg_name
  [ "$status" -eq 0 ]
  if echo "$output" | grep -q 'vgdata1'; then
    echo "VG vgdata1 still present!"
    exit 1
  fi

  # PVs should not exist (for your test disks)
  run kubectl --namespace lvminit exec "$pod" -- lvm pvs --noheadings -o pv_name
  [ "$status" -eq 0 ]
  if echo "$output" | grep -q '/dev/loop100'; then
    echo "PV /dev/loop100 still present!"
    exit 1
  fi
  if echo "$output" | grep -q '/dev/loop101'; then
    echo "PV /dev/loop101 still present!"
    exit 1
  fi
}
