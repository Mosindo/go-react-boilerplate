import { QueryClient, type InfiniteData } from "@tanstack/react-query";
import { keys } from "../api/keys";
import type { Message, RealtimeEvent } from "../api/types";
import { applyRealtimeEvent } from "../realtime/useRealtime";

const msg = (id: string): Message => ({ id, conversationId: "c1", senderId: "u2", body: id, createdAt: "2026-01-01T00:00:00Z", readAt: null });

describe("applyRealtimeEvent", () => {
  it("prepends new messages once and refreshes lists", () => {
    const qc = new QueryClient();
    const spy = jest.spyOn(qc, "invalidateQueries");
    qc.setQueryData<InfiniteData<Message[], string | undefined>>(keys.messages("c1"), { pages: [[msg("m1")]], pageParams: [undefined] });
    const ev: RealtimeEvent = { type: "message.new", data: msg("m2") };
    applyRealtimeEvent(qc, ev);
    applyRealtimeEvent(qc, ev); // duplicate delivery
    const data = qc.getQueryData<InfiniteData<Message[]>>(keys.messages("c1"));
    expect(data?.pages[0]?.map((m) => m.id)).toEqual(["m2", "m1"]);
    expect(spy).toHaveBeenCalledWith({ queryKey: keys.conversations });
  });

  it("announces matches", () => {
    const qc = new QueryClient();
    const onMatch = jest.fn();
    applyRealtimeEvent(qc, { type: "match.new", data: { id: "m", conversationId: "c", createdAt: "", user: { userId: "u", firstName: "Léa", age: 30, gender: "woman", bio: "", city: "", distanceKm: null, interests: [], photos: [] } } }, onMatch);
    expect(onMatch).toHaveBeenCalledWith("Léa");
  });
});
