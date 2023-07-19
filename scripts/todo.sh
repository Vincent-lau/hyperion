#!/usr/bin/bash

# ip route flush proto bird
# ip link list | grep cali | awk '{print $2}' | cut -c 1-15 | xargs -I {} sudo ip link delete {}
# sudo iptables-save | grep -i cali | sudo iptables -F
# sudo iptables-save | grep -i cali | sudo iptables -X
# sudo systemctl restart kubelet

# sudo sysctl net.ipv4.ip_local_port_range="15000 61000"
# sudo sysctl net.ipv4.tcp_fin_timeout=30
# sudo sysctl net.core.somaxconn=1024
# sudo sysctl net.ipv4.tcp_tw_recycle=1
# sudo sysctl net.ipv4.tcp_tw_reuse=1 

cat <<EOF | sudo tee -a /etc/sysctl.conf
net.ipv4.neigh.default.gc_thresh1 = 4096
net.ipv4.neigh.default.gc_thresh2 = 8192
net.ipv4.neigh.default.gc_thresh3 = 8192
net.ipv4.neigh.default.base_reachable_time = 86400
net.ipv4.neigh.default.gc_stale_time = 86400
EOF && \
sudo sed -i '$d' /etc/sysctl.conf && sudo sysctl -p