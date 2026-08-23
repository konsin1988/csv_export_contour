FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt /tmp/

RUN pip install -r /tmp/requirements.txt

COPY ./script.py /app/

CMD ["python", "script.py"]
