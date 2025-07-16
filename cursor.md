The tool promtool exists for Prometheus to do operations on prometheus data. One of its commands is `promtool tsdb create-blocks-from rules` which given a remote endpoint to read data from, and a rules file which indicates what the rules are, it will go and query the data from the endpoint, and write tsdb blocks containing samples for the recording rules as if they had been run for the duration. https://prometheus.io/docs/prometheus/latest/command-line/promtool/ is the docs page. The code for promtool is https://github.com/prometheus/prometheus/tree/main/cmd/promtool in case that's helpful.

How do people use this tool? Here's a blog post with how they use it: https://jessicagreben.medium.com/prometheus-fill-in-data-for-new-recording-rules-30a14ccb8467 

I would like to create a tool just like this, but it will query loki with metrics queries, save the values, and build tsdb blocks for these rules. I will then put these blocks into Grafana Cloud so that a recording rule I started running today will have historical data. A loki query is like `sum by (service_name, context_month) (count_over_time({service_name="Connections Copilot"} |= `https://connections-copilot.com` |= `"User monthly page load"` != `Googlebot` | logfmt [1m]))`, executed every minute, and the name of the recording rule is `loki_copilot_monthly_page_load_sum_by_month:1m`.

Let's build this tool in golang.
