import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi, safetyApi } from "../../api/endpoints";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { LocationCard } from "../../components/profile/LocationCard";
import { Screen } from "../../components/Screen";
import { Section, ToggleRow } from "../../components/Section";
import { LoadingState } from "../../components/States";
import { showToast } from "../../components/Toast";
import { useMyProfile } from "../../hooks/queries";
import { queryKeys } from "../../queryClient";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";

export default function PrivacyScreen() {
  const navigation = useNavigation();
  const client = useQueryClient();
  const me = useMyProfile();
  const blocks = useQuery({ queryKey: queryKeys.blocks, queryFn: async () => (await safetyApi.blocks()).blocks });
  const [saving, setSaving] = useState(false);

  const profile = me.data?.profile;
  if (!profile) return <LoadingState />;

  const update = async (isDiscoverable: boolean, showDistance: boolean) => {
    setSaving(true);
    try {
      client.setQueryData(queryKeys.myProfile, await profileApi.setPrivacy(isDiscoverable, showDistance));
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setSaving(false);
    }
  };

  const unblock = async (userId: string) => {
    try {
      await safetyApi.unblock(userId);
      await client.invalidateQueries({ queryKey: queryKeys.blocks });
      showToast("Personne débloquée.", "success");
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  return (
    <Screen scroll>
      <Header title="Confidentialité" onBack={() => navigation.goBack()} />
      <View style={styles.body}>
        <Section title="Visibilité" footer="En pause, ton profil n'apparaît plus dans Découvrir. Tes matchs et conversations restent accessibles.">
          <ToggleRow
            label="Apparaître dans Découvrir"
            value={profile.isDiscoverable}
            onChange={(v) => void update(v, profile.showDistance)}
            disabled={saving}
            testID="privacy-discoverable"
          />
          <ToggleRow
            label="Afficher la distance"
            description="Les autres voient une distance arrondie, jamais ta position."
            value={profile.showDistance}
            onChange={(v) => void update(profile.isDiscoverable, v)}
            disabled={saving}
            testID="privacy-distance"
          />
        </Section>
        <LocationCard hasLocation={profile.hasLocation} city={profile.city} />
        <Section title="Personnes bloquées">
          {blocks.isPending ? (
            <LoadingState />
          ) : (blocks.data ?? []).length === 0 ? (
            <AppText color={colors.textMuted} style={styles.empty}>
              Tu n’as bloqué personne.
            </AppText>
          ) : (
            (blocks.data ?? []).map((b) => (
              <View key={b.userId} style={styles.blockRow}>
                <AppText style={styles.flex}>{b.firstName || "Membre"}</AppText>
                <Button label="Débloquer" variant="secondary" compact onPress={() => void unblock(b.userId)} />
              </View>
            ))
          )}
        </Section>
        <AppText variant="caption" color={colors.textSubtle}>
          Bloquer quelqu’un supprime votre match et votre conversation, et vous rend mutuellement invisibles. Débloquer ne restaure pas le match.
        </AppText>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { gap: spacing.lg, marginTop: spacing.sm },
  empty: { padding: spacing.md },
  blockRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm, padding: spacing.md, borderBottomWidth: StyleSheet.hairlineWidth, borderBottomColor: colors.border },
  flex: { flex: 1 }
});
