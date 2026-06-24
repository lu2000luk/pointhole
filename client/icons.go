package main

import (
	"github.com/AllenDang/cimgui-go/imgui"
)

type IconDrawFunc func(drawList imgui.DrawList, centerX, centerY, size float32, color uint32)

func RenderIconButton(id string, drawFn IconDrawFunc, buttonSize float32, iconSize float32) bool {
	screenPos := imgui.CursorScreenPos()
	size := imgui.Vec2{X: buttonSize, Y: buttonSize}

	imgui.PushIDStr(id)
	clicked := imgui.InvisibleButton("##ibtn", size)

	isHovered := imgui.IsItemHovered()
	isActive := imgui.IsItemActive()

	var bgColor uint32
	if isActive {
		bgColor = imgui.ColorConvertFloat4ToU32(*imgui.StyleColorVec4(imgui.ColButtonActive))
	} else if isHovered {
		bgColor = imgui.ColorConvertFloat4ToU32(*imgui.StyleColorVec4(imgui.ColButtonHovered))
	} else {
		bgColor = imgui.ColorConvertFloat4ToU32(*imgui.StyleColorVec4(imgui.ColButton))
	}

	drawList := imgui.WindowDrawList()
	minPos := screenPos
	maxPos := imgui.Vec2{X: screenPos.X + size.X, Y: screenPos.Y + size.Y}
	rounding := imgui.CurrentStyle().FrameRounding()

	drawList.AddRectFilledV(minPos, maxPos, bgColor, rounding, imgui.DrawFlagsNone)

	centerX := screenPos.X + (buttonSize / 2)
	centerY := screenPos.Y + (buttonSize / 2)
	color := imgui.ColorConvertFloat4ToU32(*imgui.StyleColorVec4(imgui.ColText))

	drawFn(*drawList, centerX, centerY, iconSize, color)

	imgui.PopID()
	return clicked
}

func DrawIconRefresh(drawList imgui.DrawList, centerX, centerY, size float32, color uint32) {
	scale := size / 24.0
	thickness := 2.0 * scale
	center := imgui.Vec2{X: centerX, Y: centerY}
	radius := 9.0 * scale
	tlX := centerX - (size / 2)
	tlY := centerY - (size / 2)

	drawList.PathClear()
	drawList.PathArcToV(center, radius, 3.14159, 5.535, 16)
	drawList.PathLineTo(imgui.Vec2{X: tlX + 21*scale, Y: tlY + 8*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: tlX + 21*scale, Y: tlY + 3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 21*scale, Y: tlY + 8*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 16*scale, Y: tlY + 8*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)

	drawList.PathClear()
	drawList.PathArcToV(center, radius, 0.0, 2.39, 16)
	drawList.PathLineTo(imgui.Vec2{X: tlX + 3*scale, Y: tlY + 16*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: tlX + 8*scale, Y: tlY + 16*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 3*scale, Y: tlY + 16*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 3*scale, Y: tlY + 21*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)
}
func DrawIconUpload(drawList imgui.DrawList, centerX, centerY, size float32, color uint32) {
	scale := size / 24.0
	thickness := 2.0 * scale
	topLeftX := centerX - (size / 2)
	topLeftY := centerY - (size / 2)

	p1 := imgui.Vec2{X: topLeftX + 12*scale, Y: topLeftY + 3*scale}
	p2 := imgui.Vec2{X: topLeftX + 12*scale, Y: topLeftY + 15*scale}
	drawList.AddLineV(p1, p2, color, thickness)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 17*scale, Y: topLeftY + 8*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 12*scale, Y: topLeftY + 3*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 7*scale, Y: topLeftY + 8*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 21*scale, Y: topLeftY + 15*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 21*scale, Y: topLeftY + 19*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 19*scale, Y: topLeftY + 21*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 5*scale, Y: topLeftY + 21*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 3*scale, Y: topLeftY + 19*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 3*scale, Y: topLeftY + 15*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)
}

func DrawIconChevronLeft(drawList imgui.DrawList, centerX, centerY, size float32, color uint32) {
	scale := size / 24.0
	thickness := 2.0 * scale
	topLeftX := centerX - (size / 2)
	topLeftY := centerY - (size / 2)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 15*scale, Y: topLeftY + 18*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 9*scale, Y: topLeftY + 12*scale})
	drawList.PathLineTo(imgui.Vec2{X: topLeftX + 15*scale, Y: topLeftY + 6*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsNone, thickness)
}

func DrawIconFolderPlus(drawList imgui.DrawList, centerX, centerY, size float32, color uint32) {
	scale := size / 24.0
	thickness := 2.0 * scale
	tlX := centerX - (size / 2)
	tlY := centerY - (size / 2)

	drawList.AddLineV(imgui.Vec2{X: tlX + 12*scale, Y: tlY + 9*scale}, imgui.Vec2{X: tlX + 12*scale, Y: tlY + 17*scale}, color, thickness)
	drawList.AddLineV(imgui.Vec2{X: tlX + 8*scale, Y: tlY + 13*scale}, imgui.Vec2{X: tlX + 16*scale, Y: tlY + 13*scale}, color, thickness)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: tlX + 4*scale, Y: tlY + 20*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 20*scale, Y: tlY + 20*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 22*scale, Y: tlY + 18*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 22*scale, Y: tlY + 8*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 20*scale, Y: tlY + 6*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 12.07*scale, Y: tlY + 6*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 10.41*scale, Y: tlY + 5.1*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 9.19*scale, Y: tlY + 3.3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 8.53*scale, Y: tlY + 3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 4*scale, Y: tlY + 3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 2*scale, Y: tlY + 5*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 2*scale, Y: tlY + 18*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsClosed, thickness)
}

func DrawIconFolder(drawList imgui.DrawList, centerX, centerY, size float32, color uint32) {
	scale := size / 24.0
	thickness := 2.0 * scale
	tlX := centerX - (size / 2)
	tlY := centerY - (size / 2)

	drawList.PathClear()
	drawList.PathLineTo(imgui.Vec2{X: tlX + 4*scale, Y: tlY + 20*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 20*scale, Y: tlY + 20*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 22*scale, Y: tlY + 18*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 22*scale, Y: tlY + 8*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 20*scale, Y: tlY + 6*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 12.07*scale, Y: tlY + 6*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 10.41*scale, Y: tlY + 5.1*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 9.19*scale, Y: tlY + 3.3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 8.53*scale, Y: tlY + 3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 4*scale, Y: tlY + 3*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 2*scale, Y: tlY + 5*scale})
	drawList.PathLineTo(imgui.Vec2{X: tlX + 2*scale, Y: tlY + 18*scale})
	drawList.PathStrokeV(color, imgui.DrawFlagsClosed, thickness)
}
