# Quantum Blockchain

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Proof-of-Stake блокчейн от BilliStrom (22.06.2025)

## Особенности
- Высокая скорость транзакций (500+ TPS)
- Энергоэффективный PoS-консенсус
- Поддержка RPC-интерфейса
- Простая интеграция

## Запуск

### Требования
- Go 1.20+
- BadgerDB (автоматически установится)

### Команды
```bash
# Клонирование репозитория
git clone https://github.com/BilliStrom/quantum-blockchain
cd quantum-blockchain

# Сборка
go build -o quantum-node ./main.go

# Запуск ноды
./quantum-node --datadir ./qdata

# Запуск валидатора
./quantum-node --validator --datadir ./qdata