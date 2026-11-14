import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { MainStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { useMyProfile } from "../../state/queries";
import { spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Divider, ListRow, LoadingState, Screen, Section, Text } from "../../ui";
import { genderLabels } from "../../utils/labels";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function MeScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const { user, signOut } = useAuth();
  const { data } = useMyProfile();
  if (!data?.profile) return <LoadingState />;
  const p = data.profile;
  const filled = [p.bio, p.jobTitle, p.interests.length ? "x" : "", data.photos.length >= 3 ? "x" : "", p.hasLocation ? "x" : ""].filter(Boolean).length;
  const completion = Math.round(((filled + 1) / 6) * 100);

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <View style={[styles.ring, { borderColor: colors.primary }]}>
          <Avatar label="Votre photo principale" photo={data.photos[0]} size={104} />
        </View>
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        <Text tone="muted" variant="caption">
          {genderLabels[p.gender]} · {user?.email}
        </Text>
        <View style={[styles.meter, { backgroundColor: colors.surfaceMuted }]}>
          <View style={[styles.meterFill, { width: `${completion}%`, backgroundColor: colors.primary }]} />
        </View>
        <Text tone="subtle" variant="caption">
          Profil complété à {completion} %
        </Text>
      </View>

      <Section title="Mon profil">
        <ListRow icon="create-outline" onPress={() => navigation.navigate("EditProfile")} title="Modifier mon profil" />
        <Divider />
        <ListRow icon="images-outline" onPress={() => navigation.navigate("Photos")} title="Mes photos" value={`${data.photos.length}/6`} />
        <Divider />
        <ListRow icon="eye-outline" onPress={() => navigation.navigate("ProfileDetail", { userId: user?.id ?? "" })} title="Aperçu de mon profil" />
      </Section>
      <Section title="Rencontres">
        <ListRow icon="options-outline" onPress={() => navigation.navigate("Preferences")} title="Préférences de rencontre" />
        <Divider />
        <ListRow icon="shield-checkmark-outline" onPress={() => navigation.navigate("Privacy")} title="Confidentialité et sécurité" />
      </Section>
      <Section footer="Lueur est et restera entièrement gratuite : pas d'abonnement, pas de boost payant, pas de limite de likes.">
        <ListRow icon="log-out-outline" onPress={signOut} title="Se déconnecter" />
        <Divider />
        <ListRow destructive icon="trash-outline" onPress={() => navigation.navigate("DeleteAccount")} title="Supprimer mon compte" />
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.xs, marginBottom: spacing.xl, marginTop: spacing.sm },
  ring: { borderWidth: 3, borderRadius: 60, padding: 3, marginBottom: spacing.xs },
  meter: { width: 180, height: 6, borderRadius: 3, overflow: "hidden", marginTop: spacing.xs },
  meterFill: { height: 6 }
});
