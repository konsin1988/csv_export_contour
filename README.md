Экспорт csv файла со статистикой по Асуку. 

<p>
2) в папке /home/app/asuk/app/data/ создаём папку csv_stat со следующим содержимым:
    - файл .env:
       <br/> DB_HOST=mysql
        <br/>DB_PORT=3306
        <br/>DB_USER=root
        <br/>DB_PASSWORD=
        <br/>DB_NAME=asuk_csv_export
        <br/>BASEDIR=/app/data
        <br/>COMPANY_FILEPATH=/companies.csv
        <br/>RESULT_FILEPATH=/stat.csv
    - файл companies.csv, с компаниям. Название файла принципиально.
    - В этой же папке будет сохраняться результат обработки - stat.csv.
</p>

<p>
Команда для крона:

```docker run --rm --env-file /home/app/asuk/app/data/csv_stat/.env --network rtt -v /home/app/asuk/app/data/csv_stat/:/app/data csv-stat:0.0.1```

<br/>
Обрати внимание на название докер сети (rtt), должна совпадать с той, которая указана в компоузе, которым поднимается платформа. 
