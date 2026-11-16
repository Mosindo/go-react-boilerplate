import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMyProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme/tokens";
import { Avatar, ErrorState, Header, ListGroup, ListRow, LoadingState, Screen, Text } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { interestedInLabels } from "../../utils/format";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function MeScreen() {
  const navigation = useNavigation<Nav>();
  const me = useMyProfile();

  if (me.isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (!me.data?.profile) {
    return (
      <Screen>
        <ErrorState message={errorMessage(me.error)} onRetry={() => me.refetch()} />
      </Screen>
    );
  }
  const { profile, preferences } = me.data;
  return (
    <Screen scroll testID="me-screen">
      <Header title="Mon profil" large />
      <View style={styles.hero}>
        <Avatar uri={profile.photos[0]?.url} name={profile.firstName} size={120} ring />
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        {!profile.isDiscoverable ? (
          <View style={styles.paused}>
            <Text variant="caption" tone="accent">
              Profil en pause — tu n&apos;apparais pas dans la découverte
            </Text>
          </View>
        ) : null}
      </View>
      <View style={styles.groups}>
        <ListGroup>
          <ListRow
            icon="create-outline"
            title="Modifier mon profil"
            subtitle="Bio, métier, centres d'intérêt"
            onPress={() => navigation.navigate("EditProfile")}
            testID="me-edit-profile"
          />
          <ListRow
            icon="images-outline"
            title="Mes photos"
            subtitle={`${profile.photos.length}/6`}
            onPress={() => navigation.navigate("EditPhotos")}
            testID="me-edit-photos"
          />
          <ListRow
            icon="options-outline"
            title="Mes critères"
            subtitle={
              preferences
                ? `${preferences.interestedIn.map((g) => interestedInLabels[g].toLowerCase()).join(", ")} · ${preferences.ageMin}-${preferences.ageMax} ans · ${preferences.maxDistanceKm} km`
                : undefined
            }
            onPress={() => navigation.navigate("Preferences")}
            testID="me-preferences"
          />
        </ListGroup>
        <ListGroup>
          <ListRow
            icon="shield-checkmark-outline"
            title="Confidentialité et compte"
            subtitle="Visibilité, localisation, blocages, suppression"
            onPress={() => navigation.navigate("Settings")}
            testID="me-settings"
          />
        </ListGroup>
        <Text variant="caption" tone="subtle" align="center">
          Lueur est 100 % gratuit : aucune fonctionnalité n&apos;est payante.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.md, marginBottom: spacing.xxl },
  paused: { backgroundColor: colors.accentSoft, borderRadius: radii.pill, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  groups: { gap: spacing.lg }
});
