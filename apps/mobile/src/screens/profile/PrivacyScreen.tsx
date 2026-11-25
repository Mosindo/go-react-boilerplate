import React from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/profile";
import type { Privacy } from "../../api/types";
import { shareLocation } from "../../components/location";
import { useMyProfile, useSetMyProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Header, ListRow, LoadingState, Screen, Text, showToast } from "../../ui";
import { formatRelative } from "../../utils/format";

export function PrivacyScreen({ navigation }: AppScreenProps<"Privacy">) {
  const me = useMyProfile();
  const setMyProfile = useSetMyProfile();

  const update = useMutation({
    mutationFn: (privacy: Privacy) => profileApi.updatePrivacy(privacy),
    onSuccess: setMyProfile,
    onError: (error) => showToast(errorMessage(error), "error")
  });
  const relocate = useMutation({
    mutationFn: () => shareLocation(true),
    onSuccess: (result) => {
      if (result.ok) {
        setMyProfile(result.profile);
        showToast("Position approximative mise à jour.", "success");
      } else {
        showToast(result.reason === "denied" ? "Accès à la position refusé dans les réglages." : "Position indisponible.", "error");
      }
    }
  });

  if (!me.data) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  const privacy = update.variables ?? me.data.privacy;
  const locatedAt = me.data.profile?.locationUpdatedAt;

  return (
    <Screen scroll testID="privacy-screen">
      <Header title="Confidentialité et sécurité" onBack={navigation.goBack} />
      <View style={styles.group}>
        <ListRow
          icon="eye-off-outline"
          title="Apparaître dans Découvrir"
          subtitle="Désactivez pour faire une pause : vos matchs restent accessibles."
          toggle={{
            value: privacy.discoverable,
            onChange: (discoverable) => update.mutate({ ...privacy, discoverable }),
            disabled: update.isPending
          }}
          testID="privacy-discoverable"
        />
        <ListRow
          icon="navigate-outline"
          title="Afficher ma distance"
          subtitle="Les autres voient une distance arrondie, jamais votre position."
          toggle={{
            value: privacy.showDistance,
            onChange: (showDistance) => update.mutate({ ...privacy, showDistance }),
            disabled: update.isPending
          }}
          testID="privacy-distance"
        />
        <ListRow
          icon="locate-outline"
          title="Actualiser ma position"
          subtitle={locatedAt ? `Dernière mise à jour ${formatRelative(locatedAt)}` : undefined}
          onPress={() => relocate.mutate()}
        />
      </View>
      <View style={styles.group}>
        <ListRow
          icon="ban-outline"
          title="Personnes bloquées"
          onPress={() => navigation.navigate("BlockedUsers")}
          testID="privacy-blocked"
        />
      </View>
      <View style={styles.notes}>
        <Text variant="label" color="textSubtle">
          CE QUE NOUS PROTÉGEONS
        </Text>
        <Text variant="small" color="textMuted">
          • Votre e-mail et votre date de naissance ne sont jamais montrés.{"\n"}• Votre position est arrondie à ~1 km avant d’être
          enregistrée.{"\n"}• Les métadonnées de vos photos (dont le GPS) sont supprimées.{"\n"}• Seuls vos matchs peuvent vous écrire.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  group: {
    gap: spacing.xs,
    marginBottom: spacing.lg
  },
  notes: {
    gap: spacing.xs
  }
});
