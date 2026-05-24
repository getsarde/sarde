---
title: "Velox — Routeur HTTP Go"
description: "Routeur HTTP Go ultra-rapide, zéro allocation. Créez des APIs prêtes pour la production en quelques minutes."
layout: "splash"
hero:
  title: "Créez des APIs plus rapides avec Velox"
  tagline: "Routage zéro allocation. Chaînes de middleware. Prêt pour la production."
  image:
    light: "/img/hero-light.svg"
    dark: "/img/hero-dark.svg"
    alt: "Diagramme d'architecture Velox"
  actions:
    - text: "Commencer"
      link: "/fr/docs/guide/introduction"
      variant: "primary"
    - text: "Voir sur GitHub"
      link: "https://github.com/example/velox"
      variant: "secondary"
---

Velox est un routeur HTTP léger et haute performance pour Go. Il utilise un arbre radix pour le routage avec zéro allocation sur le chemin critique.

:::card-grid

:::card{title="Rapide" icon="zap"}
Moteur de routage zéro allocation basé sur un arbre radix. Correspondance de routes en moins d'une microseconde, même avec des milliers de routes enregistrées.
:::

:::card{title="Flexible" icon="settings"}
Système de middleware enfichable avec des chaînes composables. Ajoutez logging, auth, CORS et rate limiting en une seule ligne.
:::

:::card{title="Testé" icon="shield-check"}
Couverture de tests à 100% avec détection de conditions de course. Éprouvé en production avec des millions de requêtes par jour.
:::

:::
