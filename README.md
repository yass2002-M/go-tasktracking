# Todo Web

Application personnelle de gestion des tâches, écrite en Go. Elle permet de suivre les échéances, les priorités et le temps consacré à chaque tâche, avec une vue calendrier et un chronomètre intégré.

## Fonctionnalités

- Création, modification, suppression et prise de notes sur les tâches.
- Classement automatique en tâches du jour, en retard et autres tâches.
- Priorité réglable de 1 à 10 et planification pour aujourd'hui.
- Vue calendrier mensuelle avec navigation entre les mois.
- Suivi du temps par tâche avec un chronomètre : une seule tâche peut être chronométrée à la fois.
- Le chronomètre continue de mesurer le temps si le serveur est redémarré. Le temps est ajouté au total de la tâche à l'arrêt.
- Ajustement manuel du temps par incréments de 20 minutes.
- Checklist de sous-taches pour chaque tache : ajout, suppression et coche/décoche des éléments.
- Sauvegarde locale des données au format JSON.

## Démarrage rapide sous Windows

1. Gardez le dossier du projet complet : l'application utilise `templates` et `data` placés à côté du programme.
2. Installez Go `1.25.3` si `todoweb.exe` n'est pas déjà présent. Le lanceur le recompile automatiquement s'il manque.
3. Double-cliquez sur **Lancer Todo.bat**.
4. Le serveur démarre sur [http://localhost:8090](http://localhost:8090) et le navigateur s'ouvre automatiquement.
5. Laissez la fenêtre de commande ouverte pendant l'utilisation. Pour arrêter le serveur, revenez dans cette fenêtre et appuyez sur `Ctrl+C`.

Les exécutables compilés et les données personnelles ne sont pas versionnés. Le lanceur `.bat` garde la fenêtre ouverte afin d'afficher les éventuels messages d'erreur.

## Compiler depuis les sources

Prérequis : Go `1.25.3` ou une version compatible avec `go.mod`.

Dans PowerShell, depuis la racine du projet :

```powershell
go build -o todoweb.exe .
```

Puis lancez `Lancer Todo.bat`. Le serveur écoute sur le port `8090`.

Pour démarrer directement avec Go pendant le développement :

```powershell
go run .
```

## Utiliser le chronomètre

Sur la tâche concernée, cliquez sur **Commencer à compter**. Le compteur s'affiche et le total évolue en direct. Pour enregistrer le temps, cliquez sur **Arrêter le chrono**. Si un autre chronomètre est déjà actif, arrêtez-le avant d'en démarrer un nouveau. Les boutons `+20 min` et `−20 min` restent disponibles pour corriger le total manuellement.

## Technologies

- **Go** : serveur HTTP, logique métier et lecture/écriture des données.
- **HTML, CSS et JavaScript** : pages rendues côté serveur et interactions du chronomètre.
- **Bootstrap Icons** : icônes de l'interface, fournies par CDN.
- **JSON** : stockage local, sans base de données ni service externe.

Le projet utilise la bibliothèque standard Go ; il n'a pas de dépendance Go tierce à installer.

## Structure du dépôt

```text
.
├── main.go                 # Point d'entrée, routes HTTP et gestion des tâches
├── models/
│   └── task.go             # Modèle de tâche
├── templates/
│   ├── index.html          # Liste des tâches et chronomètre
│   └── calendar_view.html  # Calendrier et détail des tâches
├── data/
│   └── .gitkeep            # Emplacement des données locales ignorées par Git
├── Lancer Todo.bat         # Lanceur Windows avec ouverture du navigateur
├── go.mod                  # Module et version Go
└── README.md
```

Les fichiers `data/tasks.json` et `data/active_timer.json` sont créés automatiquement et ignorés par Git. Le premier contient les tâches ; le second mémorise le chronomètre actif. Sauvegardez `data/tasks.json` pour conserver vos tâches.

## Routes principales

| Méthode | Route | Rôle |
| --- | --- | --- |
| `GET` | `/` | Affiche et classe les tâches |
| `POST` | `/add`, `/edit`, `/delete` | Gère les tâches |
| `POST` | `/note`, `/update`, `/set-today` | Modifie une note, ajuste le temps ou planifie une tâche |
| `POST` | `/timer/start`, `/timer/stop` | Démarre ou arrête le chronomètre |
| `GET` | `/calendar-view` | Affiche le calendrier mensuel |

## Données et confidentialité

L'application écoute sur `localhost:8090` et conserve ses tâches dans `data/tasks.json`, dans le dossier du projet. Aucune base de données ni aucun compte en ligne n'est requis. L'accès depuis d'autres appareils du réseau local n'est pas configuré.
