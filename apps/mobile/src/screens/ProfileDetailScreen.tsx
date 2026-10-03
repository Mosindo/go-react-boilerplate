import React, { useEffect } from "react";
import { Pressable } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { errorMessage, getPublicProfile, queryKeys } from "../api/platform";
import { ProfileView } from "../components/ProfileView";
import { Screen } from "../components/Screen";
import { useSafetyMenu } from "../components/useSafetyMenu";
import { useAuth } from "../hooks/useAuth";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Text } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { name, userId } = route.params;
  const { user } = useAuth();
  const isSelf = user?.id === userId;
  const query = useQuery({ queryKey: queryKeys.publicProfile(userId), queryFn: () => getPublicProfile(userId) });
  const safety = useSafetyMenu({ name, onBlocked: () => navigation.popToTop(), userId });

  useEffect(() => {
    if (isSelf) {
      return;
    }
    navigation.setOptions({
      headerRight: () => (
        <Pressable accessibilityLabel="Report or block" accessibilityRole="button" hitSlop={12} onPress={safety.open} testID="profile-menu">
          <Text variant="heading" weight="bold">
            ⋯
          </Text>
        </Pressable>
      )
    });
  }, [isSelf, navigation, safety.open]);

  if (query.isLoading) {
    return <LoadingView fullScreen label="Loading profile…" />;
  }
  if (query.isError || !query.data) {
    const missing = (query.error as { status?: number } | null)?.status === 404;
    return (
      <ErrorView
        message={missing ? "This profile is not available anymore." : errorMessage(query.error)}
        onAction={missing ? () => navigation.goBack() : () => void query.refetch()}
        actionLabel={missing ? "Go back" : "Retry"}
      />
    );
  }
  return (
    <Screen testID="profile-detail">
      <ProfileView profile={query.data} />
      {safety.element}
    </Screen>
  );
}
