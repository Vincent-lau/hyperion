#!/usr/bin/zsh



# cmd="sudo kubeadm reset"

cmd="sudo kubeadm reset && sudo kubeadm join 128.232.80.11:6443 --token 4g1yb9.yqus1af3ziord1to --discovery-token-ca-cert-hash sha256:501dd150a1294ce588c5b5b845f65663af75a00c510751ae1d0d0ccedda93de1"
# cmd='sudo mkdir -p /etc/apt/keyrings && curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-archive-keyring.gpg && echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list && sudo apt update && sudo apt-get install --allow-change-held-packages kubelet kubeadm kubectl'
cmd="sudo mv /etc/cni/net.d/calico-kubeconfig /tmp && sudo mv /etc/cni/net.d/10-calico.conflist /tmp"
cmd="ip route flush proto bird; ip link list | grep cali | awk '{print $2}' | cut -c 1-15 | xargs -I {} sudo ip link delete {}; sudo iptables-save | grep -i cali | sudo iptables -F; sudo iptables-save | grep -i cali | sudo iptables -X; sudo systemctl restart kubelet "

#103 214 401 402 601 602 603 604
for node in 103 214 201 401 402 601 602 603 604
do
  scp ./scripts/todo.sh sl955@caelum-$node.cl.cam.ac.uk:~/
  ssh sl955@caelum-$node.cl.cam.ac.uk -t 'chmod +x todo.sh && ./todo.sh'
done

scp ./scripts/todo.sh sl955@helios.cl.cam.ac.uk:~/
ssh sl955@helios.cl.cam.ac.uk -t 'chmod +x todo.sh && ./todo.sh'