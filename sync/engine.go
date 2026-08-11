package sync

import (
	"log"

	"github.com/eugene-bert/immich-auto-albums/immich"
	"github.com/eugene-bert/immich-auto-albums/rules"
)

func Run(client *immich.Client, store *rules.Store, rule rules.Rule) error {
	assetIDs, err := client.SearchMetadata(rule)
	if err != nil {
		return err
	}

	albumID, err := client.FindAlbum(rule.AlbumName)
	if err != nil {
		return err
	}
	if albumID == "" {
		albumID, err = client.CreateAlbum(rule.AlbumName)
		if err != nil {
			return err
		}
		log.Printf("[%s] created album %q", rule.Name, rule.AlbumName)
	}

	existing, err := client.GetAlbumAssetIDs(albumID)
	if err != nil {
		return err
	}

	var newIDs []string
	for _, id := range assetIDs {
		if !existing[id] {
			newIDs = append(newIDs, id)
		}
	}

	if len(newIDs) > 0 {
		if err := client.AddAssetsToAlbum(albumID, newIDs); err != nil {
			return err
		}
		log.Printf("[%s] added %d assets (total: %d)", rule.Name, len(newIDs), len(assetIDs))
	} else {
		log.Printf("[%s] no new assets (total: %d)", rule.Name, len(assetIDs))
	}

	return store.UpdateSync(rule.ID, len(newIDs), len(assetIDs))
}
