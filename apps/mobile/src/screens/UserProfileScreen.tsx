import React from "react";
import { Alert, Dimensions, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { discoveryApi, safetyApi } from "../api/endpoints";
import { describeDistance } from "../lib/format";
import type { MainScreenProps } from "../navigation/types";
import { keys } from "../state/keys";
import { AuthImage } from "../ui/AuthImage";
import { Button } from "../ui/Button";
import { Chip } from "../ui/Chip";
import { Screen } from "../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";
import { pretty } from "../components/ProfileCard";

const WIDTH = Dimensions.get("window").width;

export function UserProfileScreen({ navigation, route }: MainScreenProps<"UserProfile">) {
  const { userId, conversationId, matchId } = route.params;
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: keys.userProfile(userId),
    queryFn: () => discoveryApi.profile(userId),
    retry: false
  });

  const refreshLists = () => {
    void queryClient.invalidateQueries({ queryKey: keys.matches });
    void queryClient.invalidateQueries({ queryKey: keys.conversations });
  };
  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      refreshLists();
      toast.show("User blocked.", "success");
      navigation.popToTop();
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });
  const unmatch = useMutation({
    mutationFn: () => discoveryApi.unmatch(matchId ?? ""),
    onSuccess: () => {
      refreshLists();
      toast.show("Unmatched.", "success");
      navigation.popToTop();
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });

  if (query.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (query.isError || !query.data) {
    return (
      <Screen>
        {query.error &&
        "status" in query.error &&
        (query.error as { status: number }).status === 404 ? (
          <EmptyState
            title="Profile unavailable"
            message="This person is no longer available."
            actionLabel="Back"
            onAction={() => navigation.goBack()}
          />
        ) : (
          <ErrorState
            message={query.error?.message ?? "Could not load this profile."}
            onRetry={() => void query.refetch()}
          />
        )}
      </Screen>
    );
  }
  const profile = query.data;
  const distance = describeDistance(profile.distanceKm);

  return (
    <Screen edges={["left", "right", "bottom"]}>
      <ScrollView>
        <ScrollView horizontal pagingEnabled showsHorizontalScrollIndicator={false}>
          {profile.photos.map((photo) => (
            <AuthImage
              key={photo.id}
              path={photo.url}
              label={`Photo of ${profile.firstName}`}
              style={{ width: WIDTH, height: WIDTH * 1.25 }}
            />
          ))}
        </ScrollView>
        <View style={styles.body}>
          <Text variant="title">
            {profile.firstName}, {profile.age}
          </Text>
          {profile.city || distance ? (
            <Text tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
          ) : null}
          {profile.bio ? <Text>{profile.bio}</Text> : null}
          <View style={styles.chips}>
            {profile.interests.map((i) => (
              <Chip key={i} label={pretty(i)} />
            ))}
          </View>
          {conversationId ? (
            <Button
              label="Message"
              onPress={() =>
                navigation.navigate("Chat", {
                  conversationId,
                  matchId,
                  peer: { userId, firstName: profile.firstName, photoId: profile.photos[0]?.id }
                })
              }
            />
          ) : null}
          {matchId ? (
            <Button
              label="Unmatch"
              variant="secondary"
              loading={unmatch.isPending}
              onPress={() =>
                Alert.alert(
                  "Unmatch?",
                  `You and ${profile.firstName} will no longer be able to message each other.`,
                  [
                    { text: "Cancel", style: "cancel" },
                    { text: "Unmatch", style: "destructive", onPress: () => unmatch.mutate() }
                  ]
                )
              }
            />
          ) : null}
          <Button
            label="Report"
            variant="secondary"
            onPress={() => navigation.navigate("Report", { userId, name: profile.firstName })}
          />
          <Button
            label="Block"
            variant="danger"
            loading={block.isPending}
            onPress={() =>
              Alert.alert(
                "Block this person?",
                "They will disappear from your matches and discovery, and cannot see your profile.",
                [
                  { text: "Cancel", style: "cancel" },
                  { text: "Block", style: "destructive", onPress: () => block.mutate() }
                ]
              )
            }
          />
        </View>
      </ScrollView>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { padding: 20, gap: 14 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
