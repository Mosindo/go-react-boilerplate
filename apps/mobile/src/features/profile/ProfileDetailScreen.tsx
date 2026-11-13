import { useQuery } from "@tanstack/react-query";
import React, { useState } from "react";
import { View } from "react-native";
import { profileApi } from "../../api/profile";
import { safetyApi } from "../../api/safety";
import type { RootScreenProps } from "../../navigation/types";
import { queryKeys } from "../../state/queryClient";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, ErrorState, Header, IconButton, LoadingState, Screen, Sheet, useToast } from "../../ui";
import { describeError } from "../../utils/errors";
import { ProfileView } from "./ProfileView";

export default function ProfileDetailScreen({ navigation, route }: RootScreenProps<"ProfileDetail">) {
  const { userId, fromChat } = route.params;
  const { spacing } = useTheme();
  const toast = useToast();
  const [menu, setMenu] = useState(false);
  const profile = useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => profileApi.getPublic(userId) });
  const name = profile.data?.firstName ?? "";

  return (
    <Screen padded={false} edges={["top"]} testID="profile-detail">
      <View style={{ paddingHorizontal: spacing.md }}>
        <Header
          back
          right={
            profile.data ? (
              <IconButton
                icon="ellipsis-horizontal"
                accessibilityLabel="Options"
                onPress={() => setMenu(true)}
                testID="profile-menu"
              />
            ) : null
          }
        />
      </View>
      {profile.isLoading ? (
        <LoadingState />
      ) : profile.data ? (
        <ProfileView
          profile={profile.data}
          footer={
            fromChat ? null : (
              <Button label="Retour à la découverte" variant="secondary" onPress={() => navigation.goBack()} />
            )
          }
        />
      ) : (
        <ErrorState
          title="Profil indisponible"
          message={describeError(profile.error)}
          actionLabel="Retour"
          onAction={() => navigation.goBack()}
        />
      )}
      <Sheet
        visible={menu}
        title={name}
        onClose={() => setMenu(false)}
        actions={[
          {
            label: `Bloquer ${name}`,
            icon: "ban-outline",
            destructive: true,
            onPress: () =>
              void safetyApi
                .block(userId)
                .then(() => {
                  toast.show(`${name} est bloqué·e.`, "success");
                  navigation.popToTop();
                })
                .catch((e) => toast.show(describeError(e), "error"))
          },
          {
            label: `Signaler ${name}`,
            icon: "flag-outline",
            destructive: true,
            testID: "profile-report",
            onPress: () => navigation.navigate("Report", { userId, firstName: name })
          }
        ]}
      />
    </Screen>
  );
}
