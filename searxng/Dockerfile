FROM searxng/searxng:2026.6.29-28d388576

COPY useragents.json /usr/local/searxng/searx/data/useragents.json
COPY utils_override.py /tmp/utils_override.py
COPY client_override.py /tmp/client_override.py

COPY check_data_cache.py /usr/local/searxng/check_data_cache.py

USER root
RUN cp /usr/local/searxng/searx/utils.py /usr/local/searxng/searx/utils.py.bak && \
    cp /usr/local/searxng/searx/network/client.py /usr/local/searxng/searx/network/client.py.bak && \
    cp /tmp/utils_override.py /usr/local/searxng/searx/utils.py && \
    cp /tmp/client_override.py /usr/local/searxng/searx/network/client.py && \
    rm /tmp/*.py && \
    find /usr/local/searxng -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true && \
    find /usr/local/searxng -name "*.pyc" -delete 2>/dev/null || true

USER searxng
