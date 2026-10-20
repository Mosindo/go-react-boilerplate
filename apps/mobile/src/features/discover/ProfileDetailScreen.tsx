import React from "react";
import { Alert, FlatList, StyleSheet, View, useWindowDimensions } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchProfile } from "../../api/profile";
import { blockUser, sendSwipe } from "../../api/social";
import { queryKeys } from "../../api/keys";
import { formatDistance } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Chip, ErrorState, LoadingState, RemoteImage, Screen, Text, useToast } from "../../ui";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ route, navigation }: Props) {
  const { userId, canAct } = route.params;
  const { width } = useWindowDimensions();
  const toast = useToast();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => fetchProfile(userId) });

  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.matches }),
        qc.invalidateQueries({ queryKey: queryKeys.conversations }),
        qc.invalidateQueries({ queryKey: queryKeys.blocks })
      ]);
      toast.show("User blocked");
      navigation.navigate("Tabs");
    },
    onError: (e) => toast.show(e instanceof Error ? e.message : "Could not block.", "error")
  });

  const swipe = useMutation({
    mutationFn: (action: "like" | "pass") => sendSwipe(userId, action),
    onSuccess: (result) => {
      void qc.invalidateQueries({ queryKey: queryKeys.matches });
      toast.show(result.matched ? "It's a match!" : "Saved");
      navigation.goBack();
    },
    onError: (e) => toast.show(e instanceof Error ? e.message : "Could not save.", "error")
  });

  if (query.isLoading) return <LoadingState />;
  if (query.isError || !query.data)
    return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  const profile = query.data;
  const photos = [...profile.photos].sort((a, b) => a.position - b.position);
  const distance = formatDistance(profile.distanceKm);

  return (
    <Screen scroll padded={false} edges={["bottom"]}>
      <FlatList
        data={photos}
        keyExtractor={(p) => p.id}
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        renderItem={({ item, index }) => (
          <RemoteImage
            url={item.url}
            style={{ width, height: width * 1.25 }}
            accessibilityLabel={`${profile.firstName} photo ${index + 1} of ${photos.length}`}
          />
        )}
      />
      <View style={styles.body}>
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        <Text tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
        {profile.bio ? <Text style={styles.section}>{profile.bio}</Text> : null}
        {profile.interests.length > 0 ? (
          <View style={[styles.chips, styles.section]}>
            {profile.interests.map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
        {canAct ? (
          <View style={styles.actions}>
            <Button
              label="Pass"
              variant="secondary"
              style={styles.flex}
              onPress={() => swipe.mutate("pass")}
              disabled={swipe.isPending}
            />
            <View style={{ width: spacing.md }} />
            <Button
              label="Like"
              style={styles.flex}
              onPress={() => swipe.mutate("like")}
              disabled={swipe.isPending}
            />
          </View>
        ) : null}
        <View style={styles.section}>
          <Button
            label="Report"
            variant="ghost"
            onPress={() => navigation.navigate("Report", { userId, name: profile.firstName })}
          />
          <Button
            label="Block"
            variant="danger"
            loading={block.isPending}
            onPress={() =>
              Alert.alert(
                `Block ${profile.firstName}?`,
                "You will no longer see each other and any conversation will be removed.",
                [
                  { text: "Cancel", style: "cancel" },
                  { text: "Block", style: "destructive", onPress: () => block.mutate() }
                ]
              )
            }
          />
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { padding: spacing.lg },
  section: { marginTop: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  actions: { flexDirection: "row", marginTop: spacing.xl },
  flex: { flex: 1 }
});
