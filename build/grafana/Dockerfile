FROM grafana/grafana:7.3.4
ADD ./provisioning /etc/grafana/provisioning
ADD ./config.ini /etc/grafana/config.ini
COPY dashboards/scheduler.json /usr/share/grafana/public/dashboards/home.json