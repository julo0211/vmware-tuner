# Guide d'Installation : VMware Tuner (Enterprise)

## Pré-requis
- OS : Debian 13 (Trixie), Debian 12, RHEL 8/9, Ubuntu 20.04+
- Architecture : AMD64 (x86_64)
- Droits : Root / Sudo

## Installation

### Option A : Environnement Air-Gap (Hors-Ligne)
C'est la méthode recommandée pour les zones sécurisées.

1. **Télécharger** le binaire `vmware-tuner-v1.1.0-linux-amd64` depuis le dépôt GitHub sur votre poste d'administration.
2. **Transférer** le fichier vers le serveur (via SCP, WinSCP, ou clé USB).
   Exemple SCP :
   ```bash
   scp path/to/vmware-tuner-v1.1.0-linux-amd64 user@serveur:/tmp/
   ```
3. **Installer** sur le serveur :
   ```bash
   cd /tmp
   chmod +x vmware-tuner-v1.1.0-linux-amd64
   sudo mv vmware-tuner-v1.1.0-linux-amd64 /usr/local/bin/vmware-tuner
   ```

### Option B : Installation Directe (Avec Internet)
```bash
wget -O vmware-tuner https://github.com/julo0211/vmware-tuner/raw/test-airgap-refactor/vmware-tuner-v1.1.0-linux-amd64
chmod +x vmware-tuner
sudo mv vmware-tuner /usr/local/bin/
```

## Utilisation

Lancer l'outil en root :
```bash
sudo vmware-tuner
```

## Fonctionnalités Clés
- **Mode Connecté/Hors-Ligne** : Détection automatique.
- **Rollback** : Sauvegardes stockées dans `/root/.vmware-tuner-backups/`.
- **Logs** : Visibles à l'écran.
