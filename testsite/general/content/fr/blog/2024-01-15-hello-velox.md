---
title: "Bonjour, Velox !"
date: 2024-01-15
description: "Présentation du routeur HTTP Go Velox."
tags: ["announcement", "open-source"]
categories: ["releases"]
params:
  featured: true
  author: "Jane Doe"
---

Nous sommes ravis d'annoncer la version alpha de Velox, un routeur HTTP haute performance pour Go.

Après des années d'utilisation de ~~DefaultServeMux de la bibliothèque standard~~ diverses solutions de routage, nous avons décidé de construire quelque chose spécialement conçu pour les serveurs API.

:::note
Velox est open source sous licence MIT. Les contributions sont les bienvenues !
:::

## Ce qui est inclus

- [x] Routage par arbre radix
- [x] Extraction de paramètres de chemin
- [x] Support des chaînes de middleware
- [ ] Support WebSocket (prévu)
- [ ] Support HTTP/3 (prévu)

## Pourquoi un autre routeur ?

La plupart des routeurs existants sacrifient soit la performance pour les fonctionnalités (Gorilla Mux), soit les fonctionnalités pour la performance. Velox vise à offrir les deux.
