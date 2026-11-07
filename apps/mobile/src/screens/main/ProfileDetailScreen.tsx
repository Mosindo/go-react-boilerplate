import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Banner, Button, ErrorState, Loading, Row, Screen } from "../../components/ui";
import { PhotoCarousel, ProfileDetails } from "../../components/ProfileView";
import { ReportSheet } from "../../components/Modals";
import { discoveryApi, profileApi, safetyApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { spacing } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";

export default function ProfileDetailScreen({
  route,
  navigation,
}: NativeStackScreenProps<MainStackParams, "ProfileDetail">) {
  const { userId, card: initial, canAct } = route.params;
  const qc = useQueryClient();
  const [reporting, setReporting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const q = useQuery({
    queryKey: ["profile-view", userId],
    queryFn: () => profileApi.view(userId),
    initialData: initial,
    staleTime: 30_000,
  });

  const afterAction = () => {
    void qc.invalidateQueries({ queryKey: ["likes"] });
    void qc.invalidateQueries({ queryKey: ["matches"] });
    void qc.invalidateQueries({ queryKey: ["conversations"] });
  };

  const swipe = useMutation({
    mutationFn: (action: "like" | "pass") => discoveryApi.swipe(userId, action),
    onSuccess: (res) => {
      afterAction();
      if (res.matched && res.match) {
        navigation.replace("Chat", {
          conversationId: res.match.conversationId,
          peerName: res.match.user.firstName,
          peerId: userId,
        });
      } else navigation.goBack();
    },
    onError: (e) => setNotice(errorMessage(e)),
  });

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      afterAction();
      void qc.invalidateQueries({ queryKey: ["blocks"] });
      navigation.popToTop();
    },
    onError: (e) => setNotice(errorMessage(e)),
  });

  if (q.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (q.isError || !q.data)
    return (
      <Screen>
        <ErrorState message="Ce profil n'est plus disponible." onRetry={() => navigation.goBack()} />
      </Screen>
    );
  const card = q.data;

  return (
    <Screen scroll padded>
      {notice ? <Banner message={notice} /> : null}
      <PhotoCarousel card={card} />
      <ProfileDetails card={card} />
      {canAct ? (
        <Row>
          <Button
            testID="detail-pass"
            title="Passer"
            variant="secondary"
            style={{ flex: 1 }}
            loading={swipe.isPending}
            onPress={() => swipe.mutate("pass")}
          />
          <Button
            testID="detail-like"
            title="Liker"
            style={{ flex: 1 }}
            loading={swipe.isPending}
            onPress={() => swipe.mutate("like")}
          />
        </Row>
      ) : null}
      <View style={{ gap: spacing.sm }}>
        <Button
          testID="block-user"
          title={`Bloquer ${card.firstName}`}
          variant="secondary"
          loading={block.isPending}
          onPress={() => block.mutate()}
        />
        <Button
          testID="report-user"
          title={`Signaler ${card.firstName}`}
          variant="ghost"
          onPress={() => setReporting(true)}
        />
      </View>
      <ReportSheet
        visible={reporting}
        userId={userId}
        name={card.firstName}
        onClose={() => setReporting(false)}
        onDone={(blocked) => {
          setReporting(false);
          afterAction();
          if (blocked) navigation.popToTop();
          else setNotice("Merci, votre signalement a été envoyé.");
        }}
      />
    </Screen>
  );
}
