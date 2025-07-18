prometheus --config.file=prometheus.yml --storage.tsdb.path=./data-30days --storage.tsdb.retention.time=60d --web.listen-address=:9091 --log.level=info
