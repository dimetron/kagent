#!/bin/bash

export target_dir=${1:-.cache/doc-query}
echo "Downloading knowledge base..."
echo "Using target directory: $target_dir"
mkdir -p $target_dir

for file in kubernetes istio argo argo-rollouts helm prometheus kgateway otel; do
  # Check if the file already exists in the cache
  url="https://doc-sqlite-db.s3.sa-east-1.amazonaws.com/$file.db"
  target_file="$target_dir/$file.db"
  if [ ! -f "$target_file" ]; then
    echo "Downloading $target_file from $url"
    curl -sLo $target_file $url || curl -sLo $target_file $url
    #sqlite3 "$target_file" "VACUUM;"
  else
    echo "File $target_file already exists, skipping download."
  fi
  #  sqlite3 "$target_file" 'SELECT count(*) FROM vec_items;'
  #  echo "Compressing $target_file"
  #  rm -f $target_file.gz"
  #  gzip -9 $target_file
  #  echo "Compressed and vacuumed $target_file.gz".
done