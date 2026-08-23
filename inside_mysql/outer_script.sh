docker exec -i mysql mkdir -p /csv_stat;
docker cp inner_script mysql:/csv_stat/;
docker cp .env mysql:/csv_stat/;
docker cp companies.csv mysql:/csv_stat/;
docker exec -i -w /csv_stat mysql /csv_stat/inner_script;
docker cp mysql:/csv_stat/stat.csv .;
docker exec -i mysql rm -rf /csv_stat;
