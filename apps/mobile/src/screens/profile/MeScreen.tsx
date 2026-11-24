import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { AppText } from "../../components/AppText";
import { Avatar } from "../../components/Avatar";
import { Header } from "../../components/Header";
import { Row, Section } from "../../components/Section";
import { Screen } from "../../components/Screen";
import { ErrorState, LoadingState } from "../../components/States";
import { useMyProfile } from "../../hooks/queries";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { genderPluralLabels } from "../../utils/format";

export default function MeScreen() {
  const navigation = useNavigation();
  const query = useMyProfile();

  if (query.isPending) return <LoadingState />;
  if (query.isError || !query.data) return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const { profile, photos, preferences, interests } = query.data;

  return (
    <Screen scroll testID="me-screen">
      <Header title="Mon profil" />
      <View style={styles.hero}>
        <Avatar url={photos[0]?.url} name={profile?.firstName ?? "?"} size={112} ring />
        <AppText variant="title">
          {profile?.firstName}
          {profile ? `, ${profile.age}` : ""}
        </AppText>
        {profile?.city ? <AppText color={colors.textMuted}>{profile.city}</AppText> : null}
        {!profile?.isDiscoverable ? (
          <AppText variant="caption" color={colors.accent}>
            Profil en pause : tu n’apparais pas dans Découvrir.
          </AppText>
        ) : null}
      </View>
      <View style={styles.sections}>
        <Section title="Profil">
          <Row icon="create-outline" label="Modifier mes infos" value={`${interests.length} centres d'intérêt`} onPress={() => navigation.navigate("EditProfile")} testID="me-edit-profile" />
          <Row icon="images-outline" label="Mes photos" value={`${photos.length}/6`} onPress={() => navigation.navigate("EditPhotos")} testID="me-edit-photos" />
          <Row
            icon="options-outline"
            label="Préférences de rencontre"
            value={`${preferences.minAge}-${preferences.maxAge} ans · ${preferences.maxDistanceKm} km`}
            onPress={() => navigation.navigate("Preferences")}
            testID="me-preferences"
          />
        </Section>
        <Section title="Sécurité">
          <Row icon="shield-checkmark-outline" label="Confidentialité & blocages" onPress={() => navigation.navigate("Privacy")} testID="me-privacy" />
          <Row icon="person-circle-outline" label="Compte" onPress={() => navigation.navigate("Account")} testID="me-account" />
        </Section>
        <AppText variant="caption" color={colors.textSubtle} align="center">
          Tu recherches : {preferences.interestedIn.map((g) => genderPluralLabels[g].toLowerCase()).join(", ")}.
        </AppText>
        <AppText variant="caption" color={colors.textSubtle} align="center">
          Lueur est 100 % gratuit — pas d’abonnement, pas de pub, pas de limite de likes.
        </AppText>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.xs, marginVertical: spacing.lg },
  sections: { gap: spacing.lg }
});
