ROOT_DIR="./"

find "$ROOT_DIR" -type d ! -path "*/.*" | while read -r dir; do
    file="$dir/test.test"
    if [ ! -f "$file" ]; then
        touch "$file"
        echo "Создан: $file"
    else
        echo "Уже существует: $file"
    fi
done
