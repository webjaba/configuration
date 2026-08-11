# configuration

Небольшой Go-пакет для загрузки обязательных параметров конфигурации из `.env` файла в пользовательскую структуру.

## Использование

```go
package main

import (
	"time"

	"github.com/webjaba/configuration"
)

type Config struct {
	App AppConfig
	DB  DBConfig
}

type AppConfig struct {
	Env   string `env:"APP_ENV"`
	Debug bool   `env:"APP_DEBUG"`
}

type DBConfig struct {
	Host    string        `env:"DB_HOST"`
	Port    int           `env:"DB_PORT"`
	Timeout time.Duration `env:"DB_TIMEOUT"`
}

func main() {
	var cfg Config

	if err := configuration.New(&cfg); err != nil {
		panic(err)
	}
}
```

`.env`:

```env
APP_ENV=local
APP_DEBUG=true
DB_HOST=localhost
DB_PORT=5432
DB_TIMEOUT=5s
```

Можно указать кастомный путь к `.env` файлу:

```go
err := configuration.New(&cfg, configuration.WithPathOption{Path: "../.env"})
```

## Правила

- Каждое экспортируемое поле, кроме вложенных структур, должно иметь `env` тег.
- Вложенные структуры парсятся рекурсивно.
- Если переменная отсутствует в `.env`, вернется ошибка.
- Если значение переменной пустое, вернется ошибка.
- Дефолтные значения не поддерживаются.
- Чтобы игнорировать поле, используйте `env:"-"`.

Поддерживаемые типы: `string`, `bool`, знаковые и беззнаковые целые числа, числа с плавающей точкой, `time.Duration` и вложенные структуры.
