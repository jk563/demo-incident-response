FROM public.ecr.aws/lambda/python:3.12-arm64

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY handler.py tools.py callback_handler.py ./
COPY prompts/ prompts/

CMD ["handler.handler"]
