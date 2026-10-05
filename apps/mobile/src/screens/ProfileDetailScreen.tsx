import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { markDeckStale } from "../lib/deckSignal";
import { queryKeys } from "../realtime/cache";
import { useState } from "react";
import { Dimensions, FlatList, StyleSheet, View } from "react-native";
import { blockUser } from "../api/platform";
import { ActionMenu, type MenuAction } from "../components/ActionMenu";
import { ReportSheet } from "../components/ReportSheet";
import { useCurrentUser } from "../hooks/useAuth";
import { useProfileCard } from "../hooks/queries";
import { confirm } from "../lib/confirm";
import { errorMessage } from "../lib/errors";
import { distanceLabel, GENDER_LABELS } from "../lib/labels";
import type { RootScreenProps } from "../navigation/types";
import { ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { IconButton, Chip, RemoteImage, Text } from "../shared/ui";
import { ScreenContainer } from "../shared/layout";
import { layout, radii, spacing } from "../theme";

/** Full profile: every photo, bio, interests. Safety actions live in the "…" menu. */
export default function ProfileDetailScreen({ route }: RootScreenProps<"ProfileDetail">) {
  const { userId } = route.params;
  const navigation = useNavigation();
  const me = useCurrentUser();
  const queryClient = useQueryClient();
  const { data: profile, isLoading, isError, error, refetch } = useProfileCard(userId);
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const isSelf = userId === me.id;

  if (isLoading) return <LoadingView />;
  if (!profile) return isError ? <ErrorView error={error} onRetry={() => void refetch()} /> : <LoadingView />;

  const width = Math.min(Dimensions.get("window").width, layout.maxContentWidth);
  const distance = distanceLabel(profile.distanceKm);

  const actions: MenuAction[] = [
    { key: "report", label: "Signaler", icon: "flag-outline", destructive: true, onPress: () => setReporting(true) },
    {
      key: "block",
      label: "Bloquer",
      icon: "ban-outline",
      destructive: true,
      onPress: async () => {
        if (!(await confirm(`Bloquer ${profile.firstName} ?`, "Vous ne vous verrez plus mutuellement.", "Bloquer", true))) return;
        try {
          await blockUser(userId);
          markDeckStale();
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          queryClient.removeQueries({ queryKey: ["messages"] });
          showSuccess(`${profile.firstName} est bloqué(e).`);
          navigation.goBack();
        } catch (e) {
          showError(errorMessage(e));
        }
      }
    }
  ];

  return (
    <ScreenContainer scroll padded={false} edges={["top", "bottom"]} testID="profile-detail">
      <View style={styles.bar}>
        <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} testID="detail-back" />
        {!isSelf ? (
          <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} testID="detail-menu" />
        ) : null}
      </View>
      <FlatList
        horizontal
        pagingEnabled
        data={profile.photos}
        keyExtractor={(p) => p.id}
        showsHorizontalScrollIndicator={false}
        style={{ width }}
        renderItem={({ item, index }) => (
          <RemoteImage
            path={item.url}
            accessibilityLabel={`Photo ${index + 1} sur ${profile.photos.length}`}
            style={{ width, height: width * 1.25 }}
          />
        )}
      />
      <View style={styles.body}>
        <Text variant="title" testID="detail-name">
          {profile.firstName}, {profile.age}
        </Text>
        <Text tone="muted">{[GENDER_LABELS[profile.gender], profile.occupation, profile.city, distance].filter(Boolean).join(" · ")}</Text>
        {profile.bio ? <Text>{profile.bio}</Text> : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
      </View>
      <ActionMenu visible={menu} actions={actions} onClose={() => setMenu(false)} />
      <ReportSheet
        visible={reporting}
        userId={userId}
        name={profile.firstName}
        onClose={() => setReporting(false)}
        onDone={(blocked) => {
          if (!blocked) return;
          markDeckStale();
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          navigation.goBack();
        }}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  bar: { flexDirection: "row", justifyContent: "space-between", width: "100%", paddingHorizontal: spacing.xs },
  body: { padding: spacing.lg, gap: spacing.md, width: "100%", borderRadius: radii.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
