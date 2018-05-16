#!/bin/bash

for i in `seq 0 1`; do
    ./bin/wsk -i action invoke allToAll -p registry 10.0.13.4:7777 -p id $i -p myNumber $(($i + 1)) -p instances 2
done
