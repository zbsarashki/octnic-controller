#!/bin/bash

#if [ "$#" -ne 2 ]; then
#  echo "Invalid arguments"
#  echo "Usage: $0 <PF-PCI-DEVICE-BDF> <comma-separated-resource-spec>"
#  exit 1
#fi

# OUTPUT_FILE=/var/run/octnic/$(echo $device_pciAddr | tr ":" "_")-config.json;
OUTPUT_FILE=/var/run/octnic/config.json

# Get PCI device BDF
PCI_DEVICE=$device_pciAddr

rm -f $OUTPUT_FILE

# Setup device
[ $numvfs != "0" ] &&  echo 0 > /sys/bus/pci/devices/$device_pciAddr/sriov_numvfs;
[ $numvfs != "0" ] && echo $numvfs > /sys/bus/pci/devices/$device_pciAddr/sriov_numvfs;
       

# Convert comma separated list into array of resources
input=$(echo $dp_resourceName | sed 's/,marvell/ marvell/g')
resources=($(echo $input | tr " " "\n"))
RESOURCE_COUNT=${#resources[@]}
if (( RESOURCE_COUNT == 0)); then
  echo "$0: no resources specified"
  exit 1
fi

for (( i = 0; i < RESOURCE_COUNT; i++ )); do
echo "Resource count $i ${resources[$i]}"
  if (( i == 0 )); then
    echo "{" > $OUTPUT_FILE
    echo "  \"resourceList\": [{" >> $OUTPUT_FILE
  fi

  # extract resource name and VF range
  parts=($(echo ${resources[$i]} | tr "#" "\n"))
  if [ "${#parts[@]}" -ne 2 ]; then
    echo "$0: Invalid resource spec: \"${resources[$i]}\""
    exit 1
  fi

  vfDevice=$(echo ${PCI_DEVICE}#${parts[1]} | sed -e 's/\(.*\),.*/\1/g') 
  vfDriver=$(echo ${PCI_DEVICE}#${parts[1]} | sed -e 's/.*,\(.*\)/\1/g') 

  echo "    \"resourceName\": \"${parts[0]}\"," >> $OUTPUT_FILE
  echo "    \"resourcePrefix\": \"marvell.com\"," >> $OUTPUT_FILE
  echo "    \"selectors\": {" >> $OUTPUT_FILE
  echo "        \"rootDevices\": [\"${vfDevice}\"]" >> $OUTPUT_FILE
  #echo "        \"rootDevices\": [\"${PCI_DEVICE}#${parts[1]}\"]" >> $OUTPUT_FILE
  echo "     }" >> $OUTPUT_FILE
  if (( i == RESOURCE_COUNT-1 )); then
    echo "   }" >> $OUTPUT_FILE
  else
    echo "   },{" >> $OUTPUT_FILE
  fi

  vfs=$(echo $vfDevice| sed -e 's/^.*#//g')
  vflist=$(for v in $(echo $vfs | tr "," " "); do
	echo $v | grep -q '-';
	if [ $? -eq 0 ] ; then
		firstVF=$(echo $v | sed -e 's/\(.*\)-.*/\1/g')
		lastVF=$(echo $v | sed -e 's/.*-\(.*\)/\1/g')
		seq $firstVF $lastVF
    	fi
	done
	)
  for VFi in $vflist; do
     vfAddr=$(ls -l /sys/bus/pci/devices/$device_pciAddr/virtfn$VFi | sed -e 's/.* -> \.\.\///g')
     crDrv=$(ls -l /sys/bus/pci/devices/$device_pciAddr/virtfn$VFi/driver | sed -e 's/.*\///g')
     vID=$(cat /sys/bus/pci/devices/$vfAddr/vendor | sed -e 's/^..//g')
     dID=$(cat /sys/bus/pci/devices/$vfAddr/device | sed -e 's/^..//g')
     [ ! -z $crDrv ] && echo $vfAddr > /sys/bus/pci/drivers/$crDrv/unbind
     #echo $vID $dID> /sys/bus/pci/drivers/$vfDriver/new_id
     echo $vfAddr > /sys/bus/pci/drivers/$vfDriver/bind
  done
done
if (( RESOURCE_COUNT )); then
  echo " ]" >> $OUTPUT_FILE
  echo "}" >> $OUTPUT_FILE
fi

[ $(ls -1 /sys/bus/pci/devices/$device_pciAddr | grep virtfn | wc -l) -eq $numvfs ] && exit 0;
exit -1;
