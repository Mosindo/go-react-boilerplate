import React from "react";
import { View } from "react-native";
import { errorMessage } from "../../api/errors";
import { useFeedback } from "../../feedback";
import { useBlocks, useMyProfile, useSavePrivacy, useUnblock } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Avatar, Button, Card, Header, LoadingState, Screen, Text, Toggle } from "../../ui";

export default function PrivacyScreen({ navigation }: AppScreenProps<"Privacy">) {
  const { toast } = useFeedback();
  const profile = useMyProfile();
  const privacy = useSavePrivacy();
  const blocks = useBlocks();
  const unblock = useUnblock();

  const update = async (change: { isDiscoverable?: boolean; showDistance?: boolean }) => {
    try {
      await privacy.mutateAsync(change);
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const p = profile.data?.profile;
  return (
    <Screen scroll testID="privacy-screen">
      <Header onBack={() => navigation.goBack()} title="Confidentialité" />
      {p ? (
        <Card>
          <Toggle
            description="Désactivé, votre profil est en pause : vous restez invisible dans la découverte. Vos matchs et conversations sont conservés."
            label="Être visible dans la découverte"
            onChange={(v) => void update({ isDiscoverable: v })}
            testID="privacy-discoverable"
            value={p.isDiscoverable}
          />
          <Toggle
            description="Seule une distance arrondie est affichée, jamais votre position."
            label="Afficher ma distance"
            onChange={(v) => void update({ showDistance: v })}
            testID="privacy-distance"
            value={p.showDistance}
          />
        </Card>
      ) : (
        <LoadingState />
      )}
      <Text variant="heading">Membres bloqués</Text>
      {blocks.isLoading ? (
        <LoadingState />
      ) : (blocks.data?.length ?? 0) === 0 ? (
        <Text color="textMuted">Vous n’avez bloqué personne.</Text>
      ) : (
        <Card>
          {blocks.data?.map((b) => (
            <View key={b.userId} style={{ flexDirection: "row", alignItems: "center", gap: spacing.sm }}>
              <Avatar name={b.firstName} size={40} uri={b.photoUrl} />
              <Text style={{ flex: 1 }} variant="bodyStrong">
                {b.firstName}
              </Text>
              <Button
                fullWidth={false}
                label="Débloquer"
                onPress={() =>
                  void unblock
                    .mutateAsync(b.userId)
                    .then(() => toast(`${b.firstName} est débloqué·e`, "success"))
                    .catch((e: unknown) => toast(errorMessage(e), "error"))
                }
                size="sm"
                variant="secondary"
              />
            </View>
          ))}
        </Card>
      )}
      <Card>
        <Text variant="bodyStrong">Rester en sécurité</Text>
        <Text color="textMuted" variant="caption">
          Ne partagez jamais d’informations bancaires. Pour un premier rendez-vous, choisissez un lieu public et prévenez un proche. Un
          comportement vous met mal à l’aise ? Signalez-le depuis le profil ou la conversation : c’est confidentiel.
        </Text>
      </Card>
    </Screen>
  );
}
