/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"slices"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) joinPresentation(chatId int64, join bool) error {
	defer func() {
		if ctx.waitConnect[chatId] != nil {
			delete(ctx.waitConnect, chatId)
		}
	}()
	connectionMode, err := ctx.binding.GetConnectionMode(chatId)
	if err != nil {
		return err
	}
	if connectionMode == ntgcalls.StreamConnection {
		if ctx.pendingConnections[chatId] != nil {
			ctx.pendingConnections[chatId].Presentation = join
		}
	} else if connectionMode == ntgcalls.RtcConnection {
		if join {
			ctx.participantsMutex.Lock()
			shouldJoin := !slices.Contains(ctx.presentations, chatId)
			ctx.participantsMutex.Unlock()

			if shouldJoin {
				ctx.participantsMutex.Lock()
				ctx.waitConnect[chatId] = make(chan error)
				ctx.participantsMutex.Unlock()

				jsonParams, err := ctx.binding.InitPresentation(chatId)
				if err != nil {
					return err
				}
				resultParams := "{\"transport\": null}"
				ctx.inputGroupCallsMutex.RLock()
				inputGroupCall := ctx.inputGroupCalls[chatId]
				ctx.inputGroupCallsMutex.RUnlock()
				callResRaw, err := ctx.App.PhoneJoinGroupCallPresentation(
					inputGroupCall,
					&tg.DataJson{
						Data: jsonParams,
					},
				)
				if err != nil {
					return err
				}
				callRes := callResRaw.(*tg.UpdatesObj)
				for _, update := range callRes.Updates {
					switch update.(type) {
					case *tg.UpdateGroupCallConnection:
						resultParams = update.(*tg.UpdateGroupCallConnection).Params.Data
					}
				}
				err = ctx.binding.Connect(
					chatId,
					resultParams,
					true,
				)
				if err != nil {
					return err
				}
				ctx.participantsMutex.Lock()
				waitChan := ctx.waitConnect[chatId]
				ctx.participantsMutex.Unlock()

				if waitChan != nil {
					<-waitChan
				}

				ctx.participantsMutex.Lock()
				if !slices.Contains(ctx.presentations, chatId) {
					ctx.presentations = append(ctx.presentations, chatId)
				}
				ctx.participantsMutex.Unlock()
			}
		} else {
			ctx.participantsMutex.Lock()
			shouldLeave := slices.Contains(ctx.presentations, chatId)
			if shouldLeave {
				ctx.presentations = stdRemove(ctx.presentations, chatId)
			}
			ctx.participantsMutex.Unlock()

			if shouldLeave {
				err = ctx.binding.StopPresentation(chatId)
				if err != nil {
					return err
				}
				ctx.inputGroupCallsMutex.RLock()
				inputGroupCall := ctx.inputGroupCalls[chatId]
				ctx.inputGroupCallsMutex.RUnlock()
				_, err = ctx.App.PhoneLeaveGroupCallPresentation(
					inputGroupCall,
				)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
